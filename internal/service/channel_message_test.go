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
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})

	msgAuthorID := uuid.NewV7()
	msgChannelID := uuid.NewV7()
	msgBody := "Hello world!"
	createdMsg, err := uut.CreateChannelMessage(
		context.Background(),
		msgAuthorID,
		msgChannelID,
		msgBody,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	selectedMsg, err := uut.GetChannelMessage(context.Background(), createdMsg.ID)
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
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})

	_, err := uut.GetChannelMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "channel message" {
		t.Fatalf("expected channel message not-found error, got %v", err)
	}
}

func TestChannelMessage_GetPage(t *testing.T) {
	channelID, firstID, secondID := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	store := &channelMessageStoreStub{channelMessages: []sqlc.ChannelMessage{
		{ID: firstID, ChannelID: channelID, Body: "first", CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true}},
		{ID: secondID, ChannelID: channelID, Body: "second", CreatedAt: pgtype.Timestamptz{Time: createdAt.Add(time.Second), Valid: true}},
		{ID: uuid.NewV7(), ChannelID: channelID, Body: "third", CreatedAt: pgtype.Timestamptz{Time: createdAt.Add(2 * time.Second), Valid: true}},
		{ID: uuid.NewV7(), ChannelID: uuid.NewV7(), Body: "other channel", CreatedAt: pgtype.Timestamptz{Time: createdAt.Add(3 * time.Second), Valid: true}},
	}}
	uut := service.NewChannelMessageService(store)

	messages, err := uut.GetChannelMessagePage(context.Background(), channelID, 2, 1)
	if err != nil {
		t.Fatalf("GetChannelMessagePage returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	expectedMessages := []service.ChannelMessage{
		{ID: secondID, ChannelID: channelID, Body: "second", CreatedAt: createdAt.Add(time.Second)},
		{ID: firstID, ChannelID: channelID, Body: "first", CreatedAt: createdAt},
	}
	if !slices.Equal(messages, expectedMessages) {
		t.Fatalf("expected messages %+v, got %+v", expectedMessages, messages)
	}
}

func TestChannelMessage_PageIncludesReplyCount(t *testing.T) {
	channelID, rootID := uuid.NewV7(), uuid.NewV7()
	store := &channelMessageStoreStub{channelMessages: []sqlc.ChannelMessage{
		{ID: rootID, ChannelID: channelID, ReplyCount: 2},
		{ID: uuid.NewV7(), ChannelID: channelID},
		{ID: uuid.NewV7(), ChannelID: channelID, ThreadRootID: &rootID},
		{ID: uuid.NewV7(), ChannelID: channelID, ThreadRootID: &rootID},
	}}
	uut := service.NewChannelMessageService(store)

	messages, err := uut.GetChannelMessagePage(context.Background(), channelID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected root and standalone message only, got %d messages", len(messages))
	}
	for _, msg := range messages {
		var want int64
		if msg.ID == rootID {
			want = 2
		}
		if msg.ReplyCount != want {
			t.Errorf("message %s: expected reply count %d, got %d", msg.ID, want, msg.ReplyCount)
		}
	}
}

func TestChannelMessage_GetThread(t *testing.T) {
	channelID, threadRootID := uuid.NewV7(), uuid.NewV7()
	firstReplyID, secondReplyID := uuid.NewV7(), uuid.NewV7()
	store := &channelMessageStoreStub{channelMessages: []sqlc.ChannelMessage{
		{ID: threadRootID, ChannelID: channelID, Body: "root"},
		{ID: firstReplyID, ChannelID: channelID, Body: "first reply", ThreadRootID: &threadRootID},
		{ID: secondReplyID, ChannelID: channelID, Body: "second reply", ThreadRootID: &threadRootID},
		{ID: uuid.NewV7(), ChannelID: channelID, Body: "not in thread"},
	}}
	uut := service.NewChannelMessageService(store)

	messages, err := uut.GetChannelThreadMessages(context.Background(), threadRootID)
	if err != nil {
		t.Fatalf("GetChannelThreadMessages returned error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 thread messages, got %d", len(messages))
	}

	expectedMessages := []service.ChannelMessage{
		{ID: firstReplyID, ChannelID: channelID, Body: "first reply", ThreadRootID: &threadRootID},
		{ID: secondReplyID, ChannelID: channelID, Body: "second reply", ThreadRootID: &threadRootID},
	}
	if !slices.Equal(messages, expectedMessages) {
		t.Fatalf("expected messages %+v, got %+v", expectedMessages, messages)
	}
}

func TestChannelMessage_GetMissingThread(t *testing.T) {
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})

	id := uuid.NewV7()
	_, err := uut.GetChannelThreadMessages(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_CreateWithMissingThreadRoot(t *testing.T) {
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})
	nonExistentRootID := uuid.NewV7()

	_, err := uut.CreateChannelMessage(
		context.Background(),
		uuid.NewV7(),
		uuid.NewV7(),
		"reply",
		&nonExistentRootID,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_Update(t *testing.T) {
	messageID, authorID, channelID := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	store := &channelMessageStoreStub{channelMessages: []sqlc.ChannelMessage{
		{ID: messageID, AuthorID: authorID, ChannelID: channelID, Body: "before", CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true}},
	}}
	uut := service.NewChannelMessageService(store)

	updated, err := uut.UpdateChannelMessage(context.Background(), messageID, "after")
	if err != nil {
		t.Fatalf("UpdateChannelMessage returned error: %v", err)
	}

	expected := service.ChannelMessage{
		ID:        messageID,
		Body:      "after",
		AuthorID:  authorID,
		ChannelID: channelID,
		CreatedAt: createdAt,
		UpdatedAt: updated.UpdatedAt,
	}
	if *updated != expected {
		t.Fatalf("expected body %+v, got %+v", expected, updated)
	}
}

func TestChannelMessage_UpdateMissing(t *testing.T) {
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})

	_, err := uut.UpdateChannelMessage(context.Background(), uuid.NewV7(), "body")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_Delete(t *testing.T) {
	messageID := uuid.NewV7()
	store := &channelMessageStoreStub{channelMessages: []sqlc.ChannelMessage{
		{ID: messageID, Body: "to delete"},
	}}
	uut := service.NewChannelMessageService(store)

	if err := uut.DeleteChannelMessage(context.Background(), messageID); err != nil {
		t.Fatalf("DeleteChannelMessage returned error: %v", err)
	}
	if len(store.channelMessages) != 0 {
		t.Fatalf("expected message to be deleted, store contains %d messages", len(store.channelMessages))
	}
}

func TestChannelMessage_DeleteMissing(t *testing.T) {
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})

	err := uut.DeleteChannelMessage(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChannelMessage_DeleteThread(t *testing.T) {
	channelID, rootID := uuid.NewV7(), uuid.NewV7()
	store := &channelMessageStoreStub{channelMessages: []sqlc.ChannelMessage{
		{ID: rootID, ChannelID: channelID, Body: "thread root"},
		{ID: uuid.NewV7(), ChannelID: channelID, Body: "thread reply", ThreadRootID: &rootID},
	}}
	uut := service.NewChannelMessageService(store)

	if err := uut.DeleteChannelThread(context.Background(), rootID); err != nil {
		t.Fatalf("DeleteChannelThread returned error: %v", err)
	}
	if len(store.channelMessages) != 0 {
		t.Fatalf("expected thread root to be deleted, store contains %d messages", len(store.channelMessages))
	}
}

func TestChannelMessage_DeleteMissingThread(t *testing.T) {
	uut := service.NewChannelMessageService(&channelMessageStoreStub{})

	err := uut.DeleteChannelThread(context.Background(), uuid.NewV7())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound service.NotFoundError
	if !errors.As(err, &notFound) || notFound.What != "channel thread" {
		t.Fatalf("expected channel thread not-found error, got %v", err)
	}
}
