package service_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

// userIdentityStoreStub stores identities and users for service tests.
type userIdentityStoreStub struct {
	identities []sqlc.UserIdentity
	users      []sqlc.User
	err        error
}

func (s *userIdentityStoreStub) CreateUserIdentity(_ context.Context, params sqlc.CreateUserIdentityParams) (sqlc.UserIdentity, error) {
	if s.err != nil {
		return sqlc.UserIdentity{}, s.err
	}
	identity := sqlc.UserIdentity{
		ID: params.ID, UserID: params.UserID, AuthSubject: params.AuthSubject, AuthIssuer: params.AuthIssuer,
		CreatedAt: pgtype.Timestamptz{Time: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC), Valid: true},
	}
	s.identities = append(s.identities, identity)
	return identity, nil
}

func (s *userIdentityStoreStub) SelectUserIdentity(_ context.Context, id uuid.UUID) (sqlc.UserIdentity, error) {
	if s.err != nil {
		return sqlc.UserIdentity{}, s.err
	}
	for _, identity := range s.identities {
		if identity.ID == id {
			return identity, nil
		}
	}
	return sqlc.UserIdentity{}, pgx.ErrNoRows
}

func (s *userIdentityStoreStub) SelectUserByIdentity(_ context.Context, params sqlc.SelectUserByIdentityParams) (sqlc.User, error) {
	if s.err != nil {
		return sqlc.User{}, s.err
	}
	for _, identity := range s.identities {
		if identity.AuthSubject == params.AuthSubject && identity.AuthIssuer == params.AuthIssuer {
			for _, user := range s.users {
				if user.ID == identity.UserID {
					return user, nil
				}
			}
		}
	}
	return sqlc.User{}, pgx.ErrNoRows
}

func (s *userIdentityStoreStub) ListUserIdentities(context.Context) ([]sqlc.UserIdentity, error) {
	return s.identities, s.err
}

func TestUserIdentity_CreateAndSelect(t *testing.T) {
	store := &userIdentityStoreStub{}
	uut := service.NewUserIdentityService(store)
	userID := uuid.New()
	created, err := uut.Create(t.Context(), userID, "subject", "issuer")
	if err != nil {
		t.Fatalf("CreateUserIdentity returned error: %v", err)
	}
	if created.ID == (uuid.UUID{}) || created.UserID != userID || created.AuthSubject != "subject" || created.AuthIssuer != "issuer" || !created.CreatedAt.Equal(store.identities[0].CreatedAt.Time) {
		t.Fatalf("unexpected identity: %+v", created)
	}
	selected, err := uut.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetUserIdentity returned error: %v", err)
	}
	if *selected != *created {
		t.Errorf("expected %+v, got %+v", created, selected)
	}
}

func TestUserIdentity_List(t *testing.T) {
	uut := service.NewUserIdentityService(&userIdentityStoreStub{})
	empty, err := uut.List(t.Context())
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-nil empty list, got %+v, %v", empty, err)
	}
	userID := uuid.New()
	var want []service.UserIdentity
	for _, subject := range []string{"first", "second"} {
		identity, err := uut.Create(t.Context(), userID, subject, "issuer")
		if err != nil {
			t.Fatalf("CreateUserIdentity returned error: %v", err)
		}
		want = append(want, *identity)
	}
	got, err := uut.List(t.Context())
	if err != nil {
		t.Fatalf("ListUserIdentities returned error: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
	if got[0].ID == got[1].ID {
		t.Error("expected distinct identity IDs")
	}
}

func TestUserIdentity_GetUserByIdentity(t *testing.T) {
	first := sqlc.User{ID: uuid.New(), Email: "ada@example.com", DisplayName: "Ada Lovelace", CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}}
	second := sqlc.User{ID: uuid.New(), Email: "grace@example.com", DisplayName: "Grace Hopper", CreatedAt: first.CreatedAt}
	store := &userIdentityStoreStub{users: []sqlc.User{first, second}, identities: []sqlc.UserIdentity{
		{UserID: first.ID, AuthSubject: "shared-subject", AuthIssuer: "issuer-one"},
		{UserID: second.ID, AuthSubject: "shared-subject", AuthIssuer: "issuer-two"},
		{UserID: first.ID, AuthSubject: "linked-subject", AuthIssuer: "issuer-one"},
	}}
	uut := service.NewUserIdentityService(store)
	for _, tc := range []struct {
		subject, issuer string
		want            sqlc.User
	}{
		{"shared-subject", "issuer-one", first},
		{"shared-subject", "issuer-two", second},
		{"linked-subject", "issuer-one", first},
	} {
		t.Run(tc.subject+"/"+tc.issuer, func(t *testing.T) {
			got, err := uut.GetUserByIdentity(t.Context(), tc.subject, tc.issuer)
			if err != nil {
				t.Fatalf("GetUserByIdentity returned error: %v", err)
			}
			want := service.User{ID: tc.want.ID, Email: tc.want.Email, DisplayName: tc.want.DisplayName, CreatedAt: tc.want.CreatedAt.Time}
			if *got != want {
				t.Errorf("expected %+v, got %+v", want, got)
			}
		})
	}
}

func TestUserIdentity_Missing(t *testing.T) {
	uut := service.NewUserIdentityService(&userIdentityStoreStub{})
	t.Run("identity", func(t *testing.T) {
		got, err := uut.Get(t.Context(), uuid.New())
		var notFound service.NotFoundError
		if got != nil || !errors.As(err, &notFound) || notFound.What != "user identity" {
			t.Fatalf("expected identity not found, got %+v, %v", got, err)
		}
	})
	t.Run("user", func(t *testing.T) {
		got, err := uut.GetUserByIdentity(t.Context(), "missing", "issuer")
		var notFound service.NotFoundError
		if got != nil || !errors.As(err, &notFound) || notFound.What != "user" {
			t.Fatalf("expected user not found, got %+v, %v", got, err)
		}
	})
}

func TestUserIdentity_StoreErrors(t *testing.T) {
	failure := errors.New("database unavailable")
	uut := service.NewUserIdentityService(&userIdentityStoreStub{err: failure})
	for _, tc := range []struct {
		name, prefix string
		run          func() error
	}{
		{"create", "create user identity", func() error {
			_, err := uut.Create(t.Context(), uuid.New(), "subject", "issuer")
			return err
		}},
		{"get", "get user identity", func() error { _, err := uut.Get(t.Context(), uuid.New()); return err }},
		{"list", "list user identities", func() error { _, err := uut.List(t.Context()); return err }},
		{"get user", "get user by identity", func() error { _, err := uut.GetUserByIdentity(t.Context(), "subject", "issuer"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if !errors.Is(err, failure) || err.Error() != tc.prefix+": "+failure.Error() {
				t.Fatalf("expected wrapped store error, got %v", err)
			}
		})
	}
}
