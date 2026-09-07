package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// ChannelMessageStore defines the data access interface for channel
// message operations
type ChannelMessageStore interface {
	CreateChannelMessage(context.Context, sqlc.CreateChannelMessageParams) (sqlc.ChannelMessage, error)
	ListChannelMessages(context.Context, sqlc.ListChannelMessagesParams) ([]sqlc.ChannelMessage, error)
	ListChannelThreadMessages(context.Context, *uuid.UUID) ([]sqlc.ChannelMessage, error)
	SelectChannelMessage(context.Context, uuid.UUID) (sqlc.ChannelMessage, error)
	UpdateChannelMessageBody(context.Context, sqlc.UpdateChannelMessageBodyParams) (sqlc.ChannelMessage, error)
	DeleteChannelMessage(context.Context, uuid.UUID) (int64, error)
}

// ChannelMessageService provides business operations for channel
// messages.
type ChannelMessageService struct {
	store ChannelMessageStore
}

// ChannelMessage is a message sent to a channel in the Waypoint domain.
type ChannelMessage struct {
	// ID is the unique identifier of the channel message.
	ID uuid.UUID
	// AuthorID is the unique identifier of the user who sent the message.
	AuthorID uuid.UUID
	// ChannelID is the unique identifier of the channel this message belongs to.
	ChannelID uuid.UUID
	// ThreadRootID is the unique identifier of the root message if this is a reply in a thread.
	ThreadRootID *uuid.UUID
	// Body is the text content of the message.
	Body string
	// CreatedAt is the time when the message was created.
	CreatedAt time.Time
	// UpdatedAt is the time when the message was last updated.
	UpdatedAt time.Time
}

func NewChannelMessageService(store ChannelMessageStore) *ChannelMessageService {
	return &ChannelMessageService{
		store: store,
	}
}

// CreateChannelMessage creates a new channel message and returns it.
func (s *ChannelMessageService) CreateChannelMessage(ctx context.Context,
	authorID uuid.UUID,
	channelID uuid.UUID,
	threadRootID *uuid.UUID,
	body string,
) (*ChannelMessage, error) {
	channelMsg, err := s.store.CreateChannelMessage(ctx, sqlc.CreateChannelMessageParams{
		ID:           uuid.NewV7(),
		AuthorID:     authorID,
		ChannelID:    channelID,
		ThreadRootID: threadRootID,
		Body:         body,
	})
	if err != nil {
		return nil, fmt.Errorf("create channel message: %w", err)
	}

	return &ChannelMessage{
		ID:           channelMsg.ID,
		AuthorID:     channelMsg.AuthorID,
		ChannelID:    channelMsg.ChannelID,
		ThreadRootID: channelMsg.ThreadRootID,
		Body:         channelMsg.Body,
		CreatedAt:    channelMsg.CreatedAt.Time,
		UpdatedAt:    channelMsg.UpdatedAt.Time,
	}, nil
}

// GetChannelMessage retrieves a channel message by its ID.
func (s *ChannelMessageService) GetChannelMessage(ctx context.Context, id uuid.UUID) (*ChannelMessage, error) {
	channelMsg, err := s.store.SelectChannelMessage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{
				What: "channel message",
			}
		}

		return nil, fmt.Errorf("get channel message: %w", err)
	}

	return &ChannelMessage{
		ID:           channelMsg.ID,
		AuthorID:     channelMsg.AuthorID,
		ChannelID:    channelMsg.ChannelID,
		ThreadRootID: channelMsg.ThreadRootID,
		Body:         channelMsg.Body,
		CreatedAt:    channelMsg.CreatedAt.Time,
		UpdatedAt:    channelMsg.UpdatedAt.Time,
	}, nil
}

// GetChannelMessagePage returns a page of channel messages for a given
// channel ID.
func (s *ChannelMessageService) GetChannelMessagePage(ctx context.Context, channelID uuid.UUID, limit int32, offset int32) ([]ChannelMessage, error) {
	channelMsgs, err := s.store.ListChannelMessages(ctx, sqlc.ListChannelMessagesParams{
		ChannelID: channelID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get channel message page: %w", err)
	}

	result := make([]ChannelMessage, 0, len(channelMsgs))
	for _, msg := range channelMsgs {
		result = append(result, ChannelMessage{
			ID:           msg.ID,
			AuthorID:     msg.AuthorID,
			ChannelID:    msg.ChannelID,
			ThreadRootID: msg.ThreadRootID,
			Body:         msg.Body,
			CreatedAt:    msg.CreatedAt.Time,
			UpdatedAt:    msg.UpdatedAt.Time,
		})
	}

	return result, nil
}

// GetChannelThreadMessages lists all messages in a thread for a given
// thread root ID pair.
func (s *ChannelMessageService) GetChannelThreadMessages(ctx context.Context, threadRootID uuid.UUID) ([]ChannelMessage, error) {
	threadMsgs, err := s.store.ListChannelThreadMessages(ctx, &threadRootID)
	if err != nil {
		return nil, fmt.Errorf("list channel thread messages: %w", err)
	}
	if len(threadMsgs) == 0 {
		return nil, NotFoundError{What: "channel thread"}
	}

	result := make([]ChannelMessage, 0, len(threadMsgs))
	for _, msg := range threadMsgs {
		result = append(result, ChannelMessage{
			ID:           msg.ID,
			AuthorID:     msg.AuthorID,
			ChannelID:    msg.ChannelID,
			ThreadRootID: msg.ThreadRootID,
			Body:         msg.Body,
			CreatedAt:    msg.CreatedAt.Time,
			UpdatedAt:    msg.UpdatedAt.Time,
		})
	}

	return result, nil
}

// UpdateChannelMessage updates the body of an existing channel message.
func (s *ChannelMessageService) UpdateChannelMessage(ctx context.Context, id uuid.UUID, body string) (*ChannelMessage, error) {
	channelMsg, err := s.store.UpdateChannelMessageBody(ctx, sqlc.UpdateChannelMessageBodyParams{
		ID:   id,
		Body: body,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "channel message"}
		}
		return nil, fmt.Errorf("update channel message: %w", err)
	}

	return &ChannelMessage{
		ID:           channelMsg.ID,
		AuthorID:     channelMsg.AuthorID,
		ChannelID:    channelMsg.ChannelID,
		ThreadRootID: channelMsg.ThreadRootID,
		Body:         channelMsg.Body,
		CreatedAt:    channelMsg.CreatedAt.Time,
		UpdatedAt:    channelMsg.UpdatedAt.Time,
	}, nil
}

// DeleteChannelMessage removes a channel message by its ID.
func (s *ChannelMessageService) DeleteChannelMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := s.store.DeleteChannelMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("delete channel message: %w", err)
	}
	if rows == 0 {
		return NotFoundError{What: "channel message"}
	}

	return nil
}

// DeleteChannelThread removes all messages in a thread given the thread root ID.
func (s *ChannelMessageService) DeleteChannelThread(ctx context.Context, threadRootID uuid.UUID) error {
	// Deleting a thread root message will also delete all of the
	// replies because of cascade delete constraints in the database.
	rows, err := s.store.DeleteChannelMessage(ctx, threadRootID)
	if err != nil {
		return fmt.Errorf("delete channel thread: %w", err)
	}
	if rows == 0 {
		return NotFoundError{What: "channel thread"}
	}

	return nil
}
