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

type channelMessageStoreStub struct {
	channelMessages []sqlc.ChannelMessage
}

func (s *channelMessageStoreStub) CreateChannelMessage(ctx context.Context, params sqlc.CreateChannelMessageParams) (sqlc.ChannelMessage, error) {
	if params.ThreadRootID != nil {
		rootExists := false
		for _, message := range s.channelMessages {
			if message.ID == *params.ThreadRootID {
				rootExists = true
				break
			}
		}
		if !rootExists {
			return sqlc.ChannelMessage{}, errors.New("foreign key violation: thread root does not exist")
		}
	}

	msg := sqlc.ChannelMessage{
		ID:           params.ID,
		Body:         params.Body,
		AuthorID:     params.AuthorID,
		ChannelID:    params.ChannelID,
		ThreadRootID: params.ThreadRootID,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	s.channelMessages = append(s.channelMessages, msg)
	return msg, nil
}

func (s *channelMessageStoreStub) ListChannelMessages(ctx context.Context, params sqlc.ListChannelMessagesParams) ([]sqlc.ChannelMessage, error) {
	var messages []sqlc.ChannelMessage
	for _, msg := range s.channelMessages {
		if msg.ChannelID == params.ChannelID && msg.ThreadRootID == nil {
			messages = append(messages, msg)
		}
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt.Time.After(messages[j].CreatedAt.Time)
	})

	start := int(params.Offset)
	if start >= len(messages) {
		return []sqlc.ChannelMessage{}, nil
	}
	end := len(messages)
	if limit := int(params.Limit); limit < end-start {
		end = start + limit
	}
	return messages[start:end], nil
}

func (s *channelMessageStoreStub) ListChannelThreadMessages(ctx context.Context, threadRootID *uuid.UUID) ([]sqlc.ChannelMessage, error) {
	var messages []sqlc.ChannelMessage

	if threadRootID == nil {
		return messages, nil
	}

	for _, msg := range s.channelMessages {
		if msg.ThreadRootID != nil && *msg.ThreadRootID == *threadRootID {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func (s *channelMessageStoreStub) SelectChannelMessage(ctx context.Context, id uuid.UUID) (sqlc.ChannelMessage, error) {
	for _, msg := range s.channelMessages {
		if msg.ID == id {
			return msg, nil
		}
	}
	return sqlc.ChannelMessage{}, pgx.ErrNoRows
}

func (s *channelMessageStoreStub) UpdateChannelMessageBody(ctx context.Context, params sqlc.UpdateChannelMessageBodyParams) (sqlc.ChannelMessage, error) {
	for i, msg := range s.channelMessages {
		if msg.ID == params.ID {
			s.channelMessages[i].Body = params.Body
			return s.channelMessages[i], nil
		}
	}
	return sqlc.ChannelMessage{}, pgx.ErrNoRows
}

func (s *channelMessageStoreStub) DeleteChannelMessage(ctx context.Context, id uuid.UUID) (int64, error) {
	for _, msg := range s.channelMessages {
		if msg.ID == id {
			remaining := make([]sqlc.ChannelMessage, 0, len(s.channelMessages)-1)
			for _, candidate := range s.channelMessages {
				if candidate.ID == id || (candidate.ThreadRootID != nil && *candidate.ThreadRootID == id) {
					continue
				}
				remaining = append(remaining, candidate)
			}
			s.channelMessages = remaining
			return 1, nil
		}
	}
	return 0, nil
}

func TestChannelMessage_CreateAndSelect(t *testing.T) {
	store := channelMessageStoreStub{}
	service := service.NewChannelMessageService(&store)

	msgAuthorID := uuid.NewV7()
	msgChannelID := uuid.NewV7()
	msgBody := "Hello world!"
	createdMsg, err := service.CreateChannelMessage(
		context.Background(),
		msgAuthorID,
		msgChannelID,
		nil,
		msgBody,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	selectedMsg, err := service.GetChannelMessage(context.Background(), createdMsg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messagesMatch := selectedMsg.ID == createdMsg.ID &&
		selectedMsg.AuthorID == createdMsg.AuthorID &&
		selectedMsg.ChannelID == createdMsg.ChannelID &&
		selectedMsg.Body == createdMsg.Body
	if !messagesMatch {
		t.Fatalf("expected %+v, got %+v", createdMsg, selectedMsg)
	}

	now := time.Now()
	if createdMsg.CreatedAt.After(now) {
		t.Fatalf("unexpected timestamp: %+v", createdMsg.CreatedAt)
	}
}

func TestChannelMessage_SelectMissing(t *testing.T) {
	messageService := service.NewChannelMessageService(&channelMessageStoreStub{})

	_, err := messageService.GetChannelMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "channel message" {
		t.Fatalf("expected channel message not-found error, got %v", err)
	}
}

func TestChannelMessage_GetPage(t *testing.T) {
	store := &channelMessageStoreStub{}
	messageService := service.NewChannelMessageService(store)
	channelID := uuid.NewV7()

	first, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), channelID, nil, "first")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}
	second, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), channelID, nil, "second")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}
	_, err = messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), channelID, nil, "third")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}

	_, err = messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), nil, "other channel")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}

	messages, err := messageService.GetChannelMessagePage(context.Background(), channelID, 2, 1)
	if err != nil {
		t.Fatalf("GetChannelMessagePage returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	expectedMessages := []service.ChannelMessage{*second, *first}
	if !slices.Equal(messages, expectedMessages) {
		t.Fatalf("expected messages %+v, got %+v", expectedMessages, messages)
	}
}

func TestChannelMessage_GetThread(t *testing.T) {
	store := &channelMessageStoreStub{}
	messageService := service.NewChannelMessageService(store)

	root, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), nil, "root")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}
	threadRootID := root.ID

	firstReply, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), &threadRootID, "first reply")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}
	secondReply, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), &threadRootID, "second reply")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}
	_, err = messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), nil, "not in thread")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}

	messages, err := messageService.GetChannelThreadMessages(context.Background(), threadRootID)
	if err != nil {
		t.Fatalf("GetChannelThreadMessages returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 thread messages, got %d", len(messages))
	}

	expectedMessages := []service.ChannelMessage{*firstReply, *secondReply}
	if !slices.Equal(messages, expectedMessages) {
		t.Fatalf("expected messages %+v, got %+v", expectedMessages, messages)
	}
}

func TestChannelMessage_GetMissingThread(t *testing.T) {
	messageService := service.NewChannelMessageService(&channelMessageStoreStub{})

	id := uuid.NewV7()
	_, err := messageService.GetChannelThreadMessages(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_CreateWithMissingThreadRoot(t *testing.T) {
	messageService := service.NewChannelMessageService(&channelMessageStoreStub{})
	nonExistentRootID := uuid.NewV7()

	_, err := messageService.CreateChannelMessage(
		context.Background(),
		uuid.NewV7(),
		uuid.NewV7(),
		&nonExistentRootID,
		"reply",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_Update(t *testing.T) {
	store := &channelMessageStoreStub{}
	messageService := service.NewChannelMessageService(store)

	created, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), nil, "before")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}

	updated, err := messageService.UpdateChannelMessage(context.Background(), created.ID, "after")
	if err != nil {
		t.Fatalf("UpdateChannelMessage returned error: %v", err)
	}

	expected := service.ChannelMessage{
		ID:        created.ID,
		Body:      "after",
		AuthorID:  created.AuthorID,
		ChannelID: created.ChannelID,
		CreatedAt: created.CreatedAt,
		UpdatedAt: updated.UpdatedAt,
	}
	if *updated != expected {
		t.Fatalf("expected body %+v, got %+v", expected, updated)
	}
}

func TestChannelMessage_UpdateMissing(t *testing.T) {
	messageService := service.NewChannelMessageService(&channelMessageStoreStub{})

	_, err := messageService.UpdateChannelMessage(context.Background(), uuid.NewV7(), "body")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_Delete(t *testing.T) {
	store := &channelMessageStoreStub{}
	messageService := service.NewChannelMessageService(store)

	created, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), nil, "to delete")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}

	if err := messageService.DeleteChannelMessage(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteChannelMessage returned error: %v", err)
	}
	if len(store.channelMessages) != 0 {
		t.Fatalf("expected message to be deleted, store contains %d messages", len(store.channelMessages))
	}
}

func TestChannelMessage_DeleteMissing(t *testing.T) {
	messageService := service.NewChannelMessageService(&channelMessageStoreStub{})

	err := messageService.DeleteChannelMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_DeleteThread(t *testing.T) {
	store := &channelMessageStoreStub{}
	messageService := service.NewChannelMessageService(store)

	root, err := messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), uuid.NewV7(), nil, "thread root")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}
	_, err = messageService.CreateChannelMessage(context.Background(), uuid.NewV7(), root.ChannelID, &root.ID, "thread reply")
	if err != nil {
		t.Fatalf("CreateChannelMessage returned error: %v", err)
	}

	if err := messageService.DeleteChannelThread(context.Background(), root.ID); err != nil {
		t.Fatalf("DeleteChannelThread returned error: %v", err)
	}
	if len(store.channelMessages) != 0 {
		t.Fatalf("expected thread root to be deleted, store contains %d messages", len(store.channelMessages))
	}
}

func TestChannelMessage_DeleteMissingThread(t *testing.T) {
	messageService := service.NewChannelMessageService(&channelMessageStoreStub{})

	err := messageService.DeleteChannelThread(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "channel thread" {
		t.Fatalf("expected channel thread not-found error, got %v", err)
	}
}
