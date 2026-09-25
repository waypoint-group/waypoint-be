//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/waypoint-group/waypoint-be/db/migrations"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
	userapi "github.com/waypoint-group/waypoint-be/internal/user"
	workspaceapi "github.com/waypoint-group/waypoint-be/internal/workspace"
	"github.com/waypoint-group/waypoint-be/tests/testlib"
)

// TestWorkspaces_Lifecycle covers membership visibility and owner-only lifecycle operations.
func TestWorkspaces_Lifecycle(t *testing.T) {
	uut := newWaypoint(t)

	owner := createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	ownerToken, memberToken := workspaceToken(t, "ada"), workspaceToken(t, "linked-ada")
	assertWorkspaceMemberships(t, uut, ownerToken)

	created := createTestWorkspace(t, uut, ownerToken, "  Engineering  ")
	if created.Name != "Engineering" || created.ID == (uuid.UUID{}) || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected workspace: %+v", created)
	}
	if len(created.Members) != 1 || created.Members[0].UserID.String() != owner.ID || created.Members[0].Role != workspaceapi.WorkspaceRoleOwner {
		t.Fatalf("expected creator as sole owner, got %+v", created.Members)
	}

	path := "/workspaces/" + created.ID.String()
	membershipPath := path + "/users/" + member.ID
	assertWorkspaceMemberships(t, uut, ownerToken, created)
	assertWorkspaceMemberships(t, uut, memberToken)

	workspaceRequest(t, uut, memberToken, http.MethodGet, path, "", http.StatusForbidden, workspaceapi.ErrForbidden)

	var added workspaceapi.WorkspaceMember
	if err := json.Unmarshal(workspaceRequest(t, uut, ownerToken, http.MethodPost, membershipPath, "", http.StatusCreated, nil), &added); err != nil {
		t.Fatal(err)
	}
	if added.UserID.String() != member.ID || added.Role != workspaceapi.WorkspaceRoleMember || added.WorkspaceID != created.ID {
		t.Fatalf("unexpected membership: %+v", added)
	}

	assertWorkspaceMemberships(t, uut, memberToken, created)

	var read workspaceapi.Workspace
	if err := json.Unmarshal(workspaceRequest(t, uut, memberToken, http.MethodGet, path, "", http.StatusOK, nil), &read); err != nil {
		t.Fatal(err)
	}
	if read.ID != created.ID || read.Name != created.Name || len(read.Members) != 2 {
		t.Fatalf("unexpected workspace: %+v", read)
	}

	workspaceRequest(t, uut, ownerToken, http.MethodPost, membershipPath, "", http.StatusConflict, workspaceapi.ErrAlreadyMember)
	workspaceRequest(t, uut, ownerToken, http.MethodDelete, path, "", http.StatusConflict, workspaceapi.ErrWorkspaceNotEmpty)
	workspaceRequest(t, uut, memberToken, http.MethodDelete, path, "", http.StatusForbidden, workspaceapi.ErrForbidden)
	workspaceRequest(t, uut, memberToken, http.MethodDelete, membershipPath, "", http.StatusForbidden, workspaceapi.ErrForbidden)
	workspaceRequest(t, uut, memberToken, http.MethodPost, path+"/users/"+owner.ID, "", http.StatusForbidden, workspaceapi.ErrForbidden)
	workspaceRequest(t, uut, ownerToken, http.MethodDelete, path+"/users/"+owner.ID, "", http.StatusConflict, workspaceapi.ErrOwnerRemoval)
	workspaceRequest(t, uut, ownerToken, http.MethodDelete, membershipPath, "", http.StatusNoContent, nil)
	assertWorkspaceMemberships(t, uut, memberToken)

	workspaceRequest(t, uut, memberToken, http.MethodGet, path, "", http.StatusForbidden, workspaceapi.ErrForbidden)
	workspaceRequest(t, uut, ownerToken, http.MethodDelete, membershipPath, "", http.StatusNotFound, workspaceapi.ErrMemberNotFound)
	workspaceRequest(t, uut, ownerToken, http.MethodDelete, path, "", http.StatusNoContent, nil)
	assertWorkspaceMemberships(t, uut, ownerToken)

	workspaceRequest(t, uut, ownerToken, http.MethodGet, path, "", http.StatusNotFound, workspaceapi.ErrNotFound)
	workspaceRequest(t, uut, ownerToken, http.MethodDelete, path, "", http.StatusNotFound, workspaceapi.ErrNotFound)
}

// TestWorkspaces_ValidationAndAuthentication rejects malformed input and untrusted or unregistered callers.
func TestWorkspaces_ValidationAndAuthentication(t *testing.T) {
	uut := newWaypoint(t)

	owner := createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	path := "/workspaces/" + created.ID.String()
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/workspaces"},
		{http.MethodGet, path},
		{http.MethodPost, path + "/users/" + owner.ID},
		{http.MethodDelete, path + "/users/" + owner.ID},
		{http.MethodDelete, path},
	} {
		workspaceRequest(t, uut, "", route.method, route.path, "{}", http.StatusUnauthorized, nil)
		workspaceRequest(t, uut, "invalid-token", route.method, route.path, "{}", http.StatusUnauthorized, nil)
		workspaceRequest(t, uut, workspaceToken(t, "missing"), route.method, route.path, "{}", http.StatusForbidden, workspaceapi.ErrForbidden)
	}

	for _, body := range []string{`{}`, `{"name":"  "}`, `null`, `[]`, `{`, `{"name":"x"} {}`, `{"name":"x","actor_id":"` + owner.ID + `"}`, `{"name":"` + strings.Repeat("x", 1<<20) + `"}`} {
		workspaceRequest(t, uut, token, http.MethodPost, "/workspaces", body, http.StatusBadRequest, nil)
	}

	for _, id := range []string{"invalid", (uuid.UUID{}).String()} {
		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			workspaceRequest(t, uut, token, method, "/workspaces/"+id, "", http.StatusBadRequest, workspaceapi.ErrInvalidID)
		}
		for _, method := range []string{http.MethodPost, http.MethodDelete} {
			workspaceRequest(t, uut, token, method, path+"/users/"+id, "", http.StatusBadRequest, workspaceapi.ErrInvalidID)
			workspaceRequest(t, uut, token, method, "/workspaces/"+id+"/users/"+owner.ID, "", http.StatusBadRequest, workspaceapi.ErrInvalidID)
		}
	}

	missing := uuid.NewV7().String()
	workspaceRequest(t, uut, token, http.MethodPost, path+"/users/"+missing, "", http.StatusNotFound, workspaceapi.ErrUserNotFound)
	workspaceRequest(t, uut, token, http.MethodDelete, path+"/users/"+missing, "", http.StatusNotFound, workspaceapi.ErrMemberNotFound)
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		workspaceRequest(t, uut, token, method, "/workspaces/"+missing+"/users/"+owner.ID, "", http.StatusNotFound, workspaceapi.ErrNotFound)
	}

	assertWorkspaceMemberships(t, uut, token, created)
}

// workspaceToken obtains an access token for an identity in the test realm.
func workspaceToken(t *testing.T, username string) string {
	t.Helper()
	token, err := testEnv.Keycloak().AccessToken(t.Context(), username, "test-password")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// workspaceRequest checks an HTTP response and returns its fully read body.
func workspaceRequest(t *testing.T, uut *testlib.TestWaypoint, token, method, path, body string, status int, expected error) []byte {
	t.Helper()

	var response *http.Response
	var err error

	if token == "" {
		response, err = uut.Request(t.Context(), method, path, strings.NewReader(body))
	} else {
		response, err = uut.RequestWithAccessToken(t.Context(), method, path, strings.NewReader(body), token)
	}
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != status {
		t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, response.StatusCode, payload)
	}
	if expected != nil && strings.TrimSpace(string(payload)) != expected.Error() {
		t.Fatalf("expected %q, got %s", expected, payload)
	}
	if status == http.StatusNoContent && len(payload) != 0 {
		t.Fatalf("expected empty body, got %s", payload)
	}
	if (status == http.StatusOK || status == http.StatusCreated) && response.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON content type")
	}

	return payload
}

// createTestWorkspace creates a workspace through the authenticated HTTP API.
func createTestWorkspace(t *testing.T, uut *testlib.TestWaypoint, token, name string) workspaceapi.Workspace {
	t.Helper()

	body, err := json.Marshal(workspaceapi.CreateWorkspaceRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	var result workspaceapi.Workspace
	if err := json.Unmarshal(workspaceRequest(t, uut, token, http.MethodPost, "/workspaces", string(body), http.StatusCreated, nil), &result); err != nil {
		t.Fatal(err)
	}

	return result
}

// assertWorkspaceMemberships keeps workspace-related /me assertions in this suite.
func assertWorkspaceMemberships(t *testing.T, uut *testlib.TestWaypoint, token string, expected ...workspaceapi.Workspace) {
	t.Helper()

	var profile userapi.MeResponse
	if err := json.Unmarshal(workspaceRequest(t, uut, token, http.MethodGet, "/me", "", http.StatusOK, nil), &profile); err != nil {
		t.Fatal(err)
	}

	if profile.WorkspaceMemberships == nil || len(profile.WorkspaceMemberships) != len(expected) {
		t.Fatalf("expected %d memberships in a JSON array, got %+v", len(expected), profile.WorkspaceMemberships)
	}

	for i, want := range expected {
		got := profile.WorkspaceMemberships[i]
		if got.WorkspaceID != want.ID.String() || got.WorkspaceName != want.Name {
			t.Fatalf("expected membership %+v, got %+v", want, got)
		}
	}
}

// TestWorkspaces_SingleOwner checks the database invariant for inserts and role changes.
func TestWorkspaces_SingleOwner(t *testing.T) {
	// HTTP does not expose role changes, so verify this constraint directly.
	postgres, err := testlib.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := postgres.Close(); err != nil {
			t.Error(err)
		}
	})

	if err := migrations.Up(postgres.URL); err != nil {
		t.Fatal(err)
	}

	conn, err := pgx.Connect(t.Context(), postgres.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := conn.Close(ctx); err != nil {
			t.Error(err)
		}
	})

	ownerID, memberID := uuid.NewV7(), uuid.NewV7()
	for _, id := range []uuid.UUID{ownerID, memberID} {
		if _, err := conn.Exec(t.Context(), "INSERT INTO users (id, email, user_name, display_name) VALUES ($1, $2, $2, 'User')", id, id.String()); err != nil {
			t.Fatal(err)
		}
	}

	workspaceID, otherID := uuid.NewV7(), uuid.NewV7()
	for _, id := range []uuid.UUID{workspaceID, otherID} {
		if _, err := conn.Exec(t.Context(), "INSERT INTO workspaces (id, name) VALUES ($1, 'Workspace')", id); err != nil {
			t.Fatal(err)
		}
		// The same user may own multiple workspaces.
		if _, err := conn.Exec(t.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'owner')", id, ownerID); err != nil {
			t.Fatal(err)
		}
	}

	_, err = conn.Exec(t.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'owner')", workspaceID, memberID)
	if !db.IsUniqueViolation(err, "workspace_members_single_owner_idx") {
		t.Fatalf("expected second owner rejected, got %v", err)
	}
	if _, err := conn.Exec(t.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'member')", workspaceID, memberID); err != nil {
		t.Fatal(err)
	}

	_, err = conn.Exec(t.Context(), "UPDATE workspace_members SET role = 'owner' WHERE workspace_id = $1 AND user_id = $2", workspaceID, memberID)
	if !db.IsUniqueViolation(err, "workspace_members_single_owner_idx") {
		t.Fatalf("expected second owner promotion rejected, got %v", err)
	}
}

// TestWorkspaces_ChannelMembershipCascade verifies channel membership constraints and cascading removal.
func TestWorkspaces_ChannelMembershipCascade(t *testing.T) {
	postgres, err := testlib.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := postgres.Close(); err != nil {
			t.Error(err)
		}
	})

	if err := migrations.Up(postgres.URL); err != nil {
		t.Fatal(err)
	}

	conn, err := pgx.Connect(t.Context(), postgres.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := conn.Close(ctx); err != nil {
			t.Error(err)
		}
	})

	ownerID, memberID, outsiderID := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, id := range []uuid.UUID{ownerID, memberID, outsiderID} {
		if _, err := conn.Exec(t.Context(), "INSERT INTO users (id, email, user_name, display_name) VALUES ($1, $2, $2, 'User')", id, id.String()); err != nil {
			t.Fatal(err)
		}
	}

	workspaceID, otherID := uuid.NewV7(), uuid.NewV7()
	channels := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	for _, id := range []uuid.UUID{workspaceID, otherID} {
		if _, err := conn.Exec(t.Context(), "INSERT INTO workspaces (id, name) VALUES ($1, 'Workspace')", id); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(t.Context(), "INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'owner'), ($1, $3, 'member')", id, ownerID, memberID); err != nil {
			t.Fatal(err)
		}
	}

	for i, channelID := range channels {
		id := workspaceID
		if i == 2 {
			id = otherID
		}
		if _, err := conn.Exec(t.Context(), "INSERT INTO channels (id, workspace_id, name) VALUES ($1, $2, $3)", channelID, id, channelID.String()); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(t.Context(), "INSERT INTO channel_members (workspace_id, channel_id, user_id) VALUES ($1, $2, $3)", id, channelID, memberID); err != nil {
			t.Fatal(err)
		}
	}

	var count int
	if err := conn.QueryRow(t.Context(), "SELECT count(*) FROM channel_members cm JOIN channels c ON c.id = cm.channel_id WHERE cm.workspace_id = c.workspace_id").Scan(&count); err != nil || count != 3 {
		t.Fatalf("expected three memberships, count=%d err=%v", count, err)
	}

	queries := sqlc.New(conn)
	if err := queries.AddChannelMember(t.Context(), sqlc.AddChannelMemberParams{WorkspaceID: workspaceID, ChannelID: channels[0], UserID: ownerID}); err != nil {
		t.Fatal(err)
	}

	// The member belongs to both workspaces, but the channel must match the supplied workspace.
	for _, tc := range []struct {
		input      sqlc.AddChannelMemberParams
		constraint string
	}{
		{sqlc.AddChannelMemberParams{WorkspaceID: otherID, ChannelID: channels[1], UserID: ownerID}, "channel_members_workspace_channel_fkey"},
		{sqlc.AddChannelMemberParams{WorkspaceID: workspaceID, ChannelID: channels[0], UserID: outsiderID}, "channel_members_workspace_member_fkey"},
	} {
		// Use a plain INSERT: ON CONFLICT DO NOTHING would skip an already existing membership.
		_, err := conn.Exec(t.Context(), "INSERT INTO channel_members (workspace_id, channel_id, user_id) VALUES ($1, $2, $3)", tc.input.WorkspaceID, tc.input.ChannelID, tc.input.UserID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" || pgErr.ConstraintName != tc.constraint {
			t.Fatalf("expected %s violation, got %v", tc.constraint, err)
		}
	}

	tx, err := conn.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), "DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2", workspaceID, memberID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(t.Context(), "SELECT count(*) FROM channel_members WHERE user_id = $1", memberID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected only other workspace membership, count=%d err=%v", count, err)
	}
	if err := tx.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(t.Context(), "SELECT count(*) FROM channel_members WHERE user_id = $1", memberID).Scan(&count); err != nil || count != 3 {
		t.Fatalf("expected cascade rollback, count=%d err=%v", count, err)
	}
	if err := queries.RemoveWorkspaceMember(t.Context(), sqlc.RemoveWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: memberID}); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(t.Context(), "SELECT count(*) FROM channel_members WHERE user_id = $1", memberID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected other workspace membership preserved, count=%d err=%v", count, err)
	}
	if err := conn.QueryRow(t.Context(), "SELECT count(*) FROM channel_members WHERE user_id = $1", ownerID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected owner's membership preserved, count=%d err=%v", count, err)
	}
	if err := queries.DeleteChannel(t.Context(), channels[0]); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(t.Context(), "SELECT count(*) FROM channel_members WHERE channel_id = $1", channels[0]).Scan(&count); err != nil || count != 0 {
		t.Fatalf("expected channel deletion cascade, count=%d err=%v", count, err)
	}
}
