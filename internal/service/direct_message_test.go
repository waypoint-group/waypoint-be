package service_test

import (
	"context"
	"errors"
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

func TestDirectMessageServiceCreatedAndSelectedDirectMessageMatch(t *testing.T) {
	store := &directMessageStoreStub{}
	messageService := service.NewDirectMessageService(store)
	authorID := uuid.NewV7()
	recipientID := uuid.NewV7()

	created, err := messageService.CreateDirectMessage(context.Background(), authorID, recipientID, "Hello world!")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}

	selected, err := messageService.GetDirectMessage(context.Background(), created.ID)
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

func TestDirectMessageServiceSelectNonExistentDirectMessageReturnsError(t *testing.T) {
	messageService := service.NewDirectMessageService(&directMessageStoreStub{})

	_, err := messageService.GetDirectMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "direct message" {
		t.Fatalf("expected direct message not-found error, got %v", err)
	}
}

func TestDirectMessageServiceGetDirectMessagePageReturnsFullPage(t *testing.T) {
	store := &directMessageStoreStub{}
	messageService := service.NewDirectMessageService(store)
	authorID := uuid.NewV7()
	recipientID := uuid.NewV7()

	first, err := messageService.CreateDirectMessage(context.Background(), authorID, recipientID, "first")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}
	second, err := messageService.CreateDirectMessage(context.Background(), recipientID, authorID, "second")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}
	third, err := messageService.CreateDirectMessage(context.Background(), authorID, recipientID, "third")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}
	_, err = messageService.CreateDirectMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), "other conversation")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}

	messages, err := messageService.GetDirectMessagePage(context.Background(), authorID, recipientID, 2, 1)
	if err != nil {
		t.Fatalf("GetDirectMessagePage returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].ID != second.ID || messages[0].Body != second.Body || messages[1].ID != first.ID || messages[1].Body != first.Body {
		t.Fatalf("unexpected messages: %+v", messages)
	}
	if third.ID == messages[0].ID || third.ID == messages[1].ID {
		t.Fatal("pagination returned a message outside the requested page")
	}
}

func TestDirectMessageServiceUpdateDirectMessageSuccess(t *testing.T) {
	store := &directMessageStoreStub{}
	messageService := service.NewDirectMessageService(store)

	created, err := messageService.CreateDirectMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), "before")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}

	updated, err := messageService.UpdateDirectMessage(context.Background(), created.ID, "after")
	if err != nil {
		t.Fatalf("UpdateDirectMessage returned error: %v", err)
	}
	if updated.ID != created.ID || updated.Body != "after" || updated.AuthorID != created.AuthorID || updated.RecipientID != created.RecipientID {
		t.Fatalf("unexpected updated message: %+v", updated)
	}
}

func TestDirectMessageServiceUpdateDirectMessageWithNonExistentMessageReturnsError(t *testing.T) {
	messageService := service.NewDirectMessageService(&directMessageStoreStub{})

	_, err := messageService.UpdateDirectMessage(context.Background(), uuid.NewV7(), "body")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDirectMessageServiceDeleteDirectMessageSuccess(t *testing.T) {
	store := &directMessageStoreStub{}
	messageService := service.NewDirectMessageService(store)

	created, err := messageService.CreateDirectMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), "to delete")
	if err != nil {
		t.Fatalf("CreateDirectMessage returned error: %v", err)
	}

	if err := messageService.DeleteDirectMessage(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteDirectMessage returned error: %v", err)
	}
	if len(store.directMessages) != 0 {
		t.Fatalf("expected message to be deleted, store contains %d messages", len(store.directMessages))
	}
}

func TestDirectMessageServiceDeleteDirectMessageWithNonExistentMessageReturnsError(t *testing.T) {
	messageService := service.NewDirectMessageService(&directMessageStoreStub{})

	err := messageService.DeleteDirectMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "direct message" {
		t.Fatalf("expected direct message not-found error, got %v", err)
	}
}
