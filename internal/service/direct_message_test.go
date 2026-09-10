package service_test

import (
	"context"
	"errors"
	"slices"
	"sort"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

type directMessageStoreStub struct {
	directMessages []sqlc.DirectMessage
}

func (s *directMessageStoreStub) CreateDirectMessage(_ context.Context, params sqlc.CreateDirectMessageParams) (sqlc.DirectMessage, error) {
	now := time.Now().Add(time.Duration(len(s.directMessages)) * time.Second)
	message := sqlc.DirectMessage{
		ID:          params.ID,
		AuthorID:    params.AuthorID,
		RecipientID: params.RecipientID,
		Body:        params.Body,
		CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
	}
	s.directMessages = append(s.directMessages, message)
	return message, nil
}

func (s *directMessageStoreStub) ListDirectMessages(_ context.Context, params sqlc.ListDirectMessagesParams) ([]sqlc.DirectMessage, error) {
	messages := make([]sqlc.DirectMessage, 0)
	for _, message := range s.directMessages {
		if sameParticipants(message.AuthorID, message.RecipientID, params.AuthorID, params.AuthorID_2) {
			messages = append(messages, message)
		}
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt.Time.After(messages[j].CreatedAt.Time)
	})

	start := int(params.Offset)
	if start >= len(messages) {
		return []sqlc.DirectMessage{}, nil
	}
	end := len(messages)
	if limit := int(params.Limit); limit < end-start {
		end = start + limit
	}
	return messages[start:end], nil
}

func (s *directMessageStoreStub) SelectDirectMessage(_ context.Context, id uuid.UUID) (sqlc.DirectMessage, error) {
	for _, message := range s.directMessages {
		if message.ID == id {
			return message, nil
		}
	}
	return sqlc.DirectMessage{}, pgx.ErrNoRows
}

func (s *directMessageStoreStub) UpdateDirectMessageBody(_ context.Context, params sqlc.UpdateDirectMessageBodyParams) (sqlc.DirectMessage, error) {
	for i, message := range s.directMessages {
		if message.ID == params.ID {
			s.directMessages[i].Body = params.Body
			return s.directMessages[i], nil
		}
	}
	return sqlc.DirectMessage{}, pgx.ErrNoRows
}

func (s *directMessageStoreStub) DeleteDirectMessage(_ context.Context, id uuid.UUID) (int64, error) {
	for i, message := range s.directMessages {
		if message.ID == id {
			s.directMessages = append(s.directMessages[:i], s.directMessages[i+1:]...)
			return 1, nil
		}
	}
	return 0, nil
}

func sameParticipants(authorID, recipientID, otherAuthorID, otherRecipientID uuid.UUID) bool {
	return (authorID == otherAuthorID && recipientID == otherRecipientID) ||
		(authorID == otherRecipientID && recipientID == otherAuthorID)
}

func TestDirectMessage_CreateAndSelect(t *testing.T) {
	uut := service.NewDirectMessageService(&directMessageStoreStub{})
	authorID := uuid.NewV7()
	recipientID := uuid.NewV7()

	created, err := uut.CreateDirectMessage(context.Background(), authorID, recipientID, "Hello world!")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}

	selected, err := uut.GetDirectMessage(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetDirectMessage returned error: %v", err)
	}
	if selected.ID != created.ID || selected.AuthorID != authorID || selected.RecipientID != recipientID || selected.Body != created.Body {
		t.Fatalf("expected %+v, got %+v", created, selected)
	}
	if created.CreatedAt.After(time.Now()) {
		t.Fatalf("unexpected timestamp: %v", created.CreatedAt)
	}
}

func TestDirectMessage_SelectMissing(t *testing.T) {
	uut := service.NewDirectMessageService(&directMessageStoreStub{})

	_, err := uut.GetDirectMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "direct message" {
		t.Fatalf("expected direct message not-found error, got %v", err)
	}
}

func TestDirectMessage_GetPage(t *testing.T) {
	authorID, recipientID := uuid.NewV7(), uuid.NewV7()
	firstID, secondID, thirdID := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	store := &directMessageStoreStub{directMessages: []sqlc.DirectMessage{
		{ID: firstID, AuthorID: authorID, RecipientID: recipientID, Body: "first", CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true}},
		{ID: secondID, AuthorID: recipientID, RecipientID: authorID, Body: "second", CreatedAt: pgtype.Timestamptz{Time: createdAt.Add(time.Second), Valid: true}},
		{ID: thirdID, AuthorID: authorID, RecipientID: recipientID, Body: "third", CreatedAt: pgtype.Timestamptz{Time: createdAt.Add(2 * time.Second), Valid: true}},
		{ID: uuid.NewV7(), AuthorID: uuid.NewV7(), RecipientID: uuid.NewV7(), Body: "other conversation", CreatedAt: pgtype.Timestamptz{Time: createdAt.Add(3 * time.Second), Valid: true}},
	}}
	uut := service.NewDirectMessageService(store)

	messages, err := uut.GetDirectMessagePage(t.Context(), authorID, recipientID, 2, 1)
	if err != nil {
		t.Fatalf("GetDirectMessagePage returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	expectedMessages := []service.DirectMessage{
		{ID: secondID, AuthorID: recipientID, RecipientID: authorID, Body: "second", CreatedAt: createdAt.Add(time.Second)},
		{ID: firstID, AuthorID: authorID, RecipientID: recipientID, Body: "first", CreatedAt: createdAt},
	}
	if !slices.Equal(messages, expectedMessages) {
		t.Fatalf("expected messages %+v, got %+v", expectedMessages, messages)
	}
}

func TestDirectMessage_Update(t *testing.T) {
	messageID, authorID, recipientID := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	store := &directMessageStoreStub{directMessages: []sqlc.DirectMessage{
		{ID: messageID, AuthorID: authorID, RecipientID: recipientID, Body: "before"},
	}}
	uut := service.NewDirectMessageService(store)

	updated, err := uut.UpdateDirectMessage(t.Context(), messageID, "after")
	if err != nil {
		t.Fatalf("UpdateDirectMessage returned error: %v", err)
	}
	if updated.ID != messageID || updated.Body != "after" || updated.AuthorID != authorID || updated.RecipientID != recipientID {
		t.Fatalf("unexpected updated message: %+v", updated)
	}
}

func TestDirectMessage_UpdateMissing(t *testing.T) {
	uut := service.NewDirectMessageService(&directMessageStoreStub{})

	_, err := uut.UpdateDirectMessage(t.Context(), uuid.NewV7(), "body")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDirectMessage_Delete(t *testing.T) {
	messageID := uuid.NewV7()
	store := &directMessageStoreStub{directMessages: []sqlc.DirectMessage{
		{ID: messageID, Body: "to delete"},
	}}
	uut := service.NewDirectMessageService(store)

	if err := uut.DeleteDirectMessage(t.Context(), messageID); err != nil {
		t.Fatalf("DeleteDirectMessage returned error: %v", err)
	}
	if len(store.directMessages) != 0 {
		t.Fatalf("expected message to be deleted, store contains %d messages", len(store.directMessages))
	}
}

func TestDirectMessage_DeleteMissing(t *testing.T) {
	uut := service.NewDirectMessageService(&directMessageStoreStub{})

	err := uut.DeleteDirectMessage(t.Context(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "direct message" {
		t.Fatalf("expected direct message not-found error, got %v", err)
	}
}
