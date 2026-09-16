package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// MessagingService manages direct messages and channel messages.
type MessagingService struct {
	database *db.Database
}

func NewMessagingService(database *db.Database) *MessagingService {
	return &MessagingService{database: database}
}

// DirectMessage contains the input for sending a message to another user.
type DirectMessage struct {
	// AuthorID is the unique identifier of the user who sent the message.
	AuthorID uuid.UUID
	// RecipientID is the unique identifier of the user who receives the message.
	RecipientID uuid.UUID
	// Body is the text content of the message.
	Body string
}

// SentDirectMessage contains a persisted direct message and its timestamps.
type SentDirectMessage struct {
	// ID is the unique identifier of the direct message.
	ID uuid.UUID
	// AuthorID is the unique identifier of the user who sent the message.
	AuthorID uuid.UUID
	// RecipientID is the unique identifier of the user who receives the message.
	RecipientID uuid.UUID
	// Body is the text content of the message.
	Body string
	// CreatedAt is the time when the message was created.
	CreatedAt time.Time
	// UpdatedAt is the time when the message was last updated.
	UpdatedAt time.Time
}

// SendDirectMessage stores a nonempty message between distinct users.
func (s *MessagingService) SendDirectMessage(ctx context.Context, dm DirectMessage) (*SentDirectMessage, error) {
	if dm.Body == "" {
		return nil, InvalidInputError{What: "message body cannot be empty"}
	}
	if dm.AuthorID == dm.RecipientID {
		return nil, InvalidInputError{What: "author and recipient cannot be the same"}
	}

	sent, err := s.database.CreateDirectMessage(ctx, sqlc.CreateDirectMessageParams{
		ID:          uuid.NewV7(),
		AuthorID:    dm.AuthorID,
		RecipientID: dm.RecipientID,
		Body:        dm.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("create direct message: %w", err)
	}

	return &SentDirectMessage{
		ID:          sent.ID,
		AuthorID:    sent.AuthorID,
		RecipientID: sent.RecipientID,
		Body:        sent.Body,
		CreatedAt:   sent.CreatedAt.Time,
		UpdatedAt:   sent.UpdatedAt.Time,
	}, nil
}

// UpdateDirectMessage replaces a message body with nonempty text.
func (s *MessagingService) UpdateDirectMessage(ctx context.Context, id uuid.UUID, newBody string) (*SentDirectMessage, error) {
	if newBody == "" {
		return nil, InvalidInputError{What: "message body cannot be empty"}
	}

	updated, err := s.database.UpdateDirectMessageBody(ctx, sqlc.UpdateDirectMessageBodyParams{
		ID:   id,
		Body: newBody,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "direct message"}
		}
		return nil, fmt.Errorf("update direct message: %w", err)
	}

	return &SentDirectMessage{
		ID:          updated.ID,
		AuthorID:    updated.AuthorID,
		RecipientID: updated.RecipientID,
		Body:        updated.Body,
		CreatedAt:   updated.CreatedAt.Time,
		UpdatedAt:   updated.UpdatedAt.Time,
	}, nil
}

// ReadSingleDirectMessage retrieves a direct message by ID.
func (s *MessagingService) ReadSingleDirectMessage(
	ctx context.Context,
	id uuid.UUID,
) (*SentDirectMessage, error) {
	msg, err := s.database.SelectDirectMessage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "direct message"}
		}
		return nil, fmt.Errorf("read single direct message: %w", err)
	}

	return &SentDirectMessage{
		ID:          msg.ID,
		AuthorID:    msg.AuthorID,
		RecipientID: msg.RecipientID,
		Body:        msg.Body,
		CreatedAt:   msg.CreatedAt.Time,
		UpdatedAt:   msg.UpdatedAt.Time,
	}, nil
}

// ReadDirectMessagePage returns messages between two users, newest first.
func (s *MessagingService) ReadDirectMessagePage(
	ctx context.Context,
	userA uuid.UUID,
	userB uuid.UUID,
	limit uint32,
	offset uint32,
) ([]SentDirectMessage, error) {
	if limit > math.MaxInt32 {
		return nil, InvalidInputError{What: fmt.Sprintf("limit (%v) exceeds maximum allowed value", limit)}
	}
	if offset > math.MaxInt32 {
		return nil, InvalidInputError{What: fmt.Sprintf("offset (%v) exceeds maximum allowed value", offset)}
	}
	if userA == userB {
		return nil, InvalidInputError{What: "users must be different"}
	}

	messages, err := s.database.ListDirectMessages(ctx, sqlc.ListDirectMessagesParams{
		AuthorID:   userA,
		AuthorID_2: userB,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list direct messages: %w", err)
	}

	result := make([]SentDirectMessage, len(messages))
	for i, m := range messages {
		result[i] = SentDirectMessage{
			ID:          m.ID,
			AuthorID:    m.AuthorID,
			RecipientID: m.RecipientID,
			Body:        m.Body,
			CreatedAt:   m.CreatedAt.Time,
			UpdatedAt:   m.UpdatedAt.Time,
		}
	}

	return result, nil
}

// DeleteDirectMessage removes a message, returning NotFoundError if absent.
func (s *MessagingService) DeleteDirectMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := s.database.DeleteDirectMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("delete direct message: %w", err)
	}
	if rows == 0 {
		return NotFoundError{What: "direct message"}
	}

	return nil
}

// ChannelMessage contains the input for sending a channel message or reply.
type ChannelMessage struct {
	// AuthorID is the unique identifier of the user who sent the message.
	AuthorID uuid.UUID
	// ChannelID is the unique identifier of the channel this message belongs to.
	ChannelID uuid.UUID
	// Body is the text content of the message.
	Body string
	// ThreadRootID is the unique identifier of the root message if this is
	// a reply in a thread.
	ThreadRootID *uuid.UUID
}

// SentChannelMessage contains a persisted channel message and its thread metadata.
type SentChannelMessage struct {
	// ID is the unique identifier of the channel message.
	ID uuid.UUID
	// AuthorID is the unique identifier of the user who sent the message.
	AuthorID uuid.UUID
	// ChannelID is the unique identifier of the channel this message belongs to.
	ChannelID uuid.UUID
	// Body is the text content of the message.
	Body string
	// ThreadRootID is the unique identifier of the root message if this is
	// a reply in a thread.
	ThreadRootID *uuid.UUID
	// ReplyCount is the number of replies in the thread. Zero if this is not
	// a thread root.
	ReplyCount int64
	// CreatedAt is the time when the message was created.
	CreatedAt time.Time
	// UpdatedAt is the time when the message was last updated.
	UpdatedAt time.Time
}

// SendChannelMessage stores a nonempty channel message or thread reply.
func (s *MessagingService) SendChannelMessage(ctx context.Context, cm ChannelMessage) (*SentChannelMessage, error) {
	if cm.Body == "" {
		return nil, InvalidInputError{What: "message body cannot be empty"}
	}

	sent, err := s.database.CreateChannelMessage(ctx, sqlc.CreateChannelMessageParams{
		ID:           uuid.NewV7(),
		AuthorID:     cm.AuthorID,
		ChannelID:    cm.ChannelID,
		Body:         cm.Body,
		ThreadRootID: cm.ThreadRootID,
	})
	if err != nil {
		return nil, fmt.Errorf("create channel message: %w", err)
	}

	return &SentChannelMessage{
		ID:           sent.ID,
		AuthorID:     sent.AuthorID,
		ChannelID:    sent.ChannelID,
		Body:         sent.Body,
		ThreadRootID: sent.ThreadRootID,
		ReplyCount:   sent.ReplyCount,
		CreatedAt:    sent.CreatedAt.Time,
		UpdatedAt:    sent.UpdatedAt.Time,
	}, nil
}

// UpdateChannelMessage replaces a message body with nonempty text.
func (s *MessagingService) UpdateChannelMessage(ctx context.Context, id uuid.UUID, newBody string) (*SentChannelMessage, error) {
	if newBody == "" {
		return nil, InvalidInputError{What: "message body cannot be empty"}
	}

	updated, err := s.database.UpdateChannelMessageBody(ctx, sqlc.UpdateChannelMessageBodyParams{
		ID:   id,
		Body: newBody,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "channel message"}
		}
		return nil, fmt.Errorf("update channel message: %w", err)
	}

	return &SentChannelMessage{
		ID:           updated.ID,
		AuthorID:     updated.AuthorID,
		ChannelID:    updated.ChannelID,
		Body:         updated.Body,
		ThreadRootID: updated.ThreadRootID,
		ReplyCount:   updated.ReplyCount,
		CreatedAt:    updated.CreatedAt.Time,
		UpdatedAt:    updated.UpdatedAt.Time,
	}, nil
}

// ReadSingleChannelMessage retrieves a channel message or reply by ID.
func (s *MessagingService) ReadSingleChannelMessage(
	ctx context.Context,
	id uuid.UUID,
) (*SentChannelMessage, error) {
	msg, err := s.database.SelectChannelMessage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "channel message"}
		}
		return nil, fmt.Errorf("read single channel message: %w", err)
	}

	return &SentChannelMessage{
		ID:           msg.ID,
		AuthorID:     msg.AuthorID,
		ChannelID:    msg.ChannelID,
		Body:         msg.Body,
		ThreadRootID: msg.ThreadRootID,
		ReplyCount:   msg.ReplyCount,
		CreatedAt:    msg.CreatedAt.Time,
		UpdatedAt:    msg.UpdatedAt.Time,
	}, nil
}

// ReadChannelMessagePage returns top-level channel messages, newest first.
func (s *MessagingService) ReadChannelMessagePage(
	ctx context.Context,
	channelID uuid.UUID,
	limit uint32,
	offset uint32,
) ([]SentChannelMessage, error) {
	if limit > math.MaxInt32 {
		return nil, InvalidInputError{What: fmt.Sprintf("limit (%v) exceeds maximum allowed value", limit)}
	}
	if offset > math.MaxInt32 {
		return nil, InvalidInputError{What: fmt.Sprintf("offset (%v) exceeds maximum allowed value", offset)}
	}

	messages, err := s.database.ListChannelMessages(ctx, sqlc.ListChannelMessagesParams{
		ChannelID: channelID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list channel messages: %w", err)
	}

	result := make([]SentChannelMessage, len(messages))
	for i, m := range messages {
		result[i] = SentChannelMessage{
			ID:           m.ID,
			AuthorID:     m.AuthorID,
			ChannelID:    m.ChannelID,
			Body:         m.Body,
			ThreadRootID: m.ThreadRootID,
			ReplyCount:   m.ReplyCount,
			CreatedAt:    m.CreatedAt.Time,
			UpdatedAt:    m.UpdatedAt.Time,
		}
	}

	return result, nil
}

// ReadChannelThreadMessages returns replies ordered by creation time.
// It returns NotFoundError when the thread has no replies.
func (s *MessagingService) ReadChannelThreadMessages(ctx context.Context, threadRootID uuid.UUID) ([]SentChannelMessage, error) {
	messages, err := s.database.ListChannelThreadMessages(ctx, &threadRootID)
	if err != nil {
		return nil, fmt.Errorf("read channel thread messages: %w", err)
	}
	if len(messages) == 0 {
		return nil, NotFoundError{What: "channel thread"}
	}

	result := make([]SentChannelMessage, len(messages))
	for i, m := range messages {
		result[i] = SentChannelMessage{
			ID:           m.ID,
			AuthorID:     m.AuthorID,
			ChannelID:    m.ChannelID,
			Body:         m.Body,
			ThreadRootID: m.ThreadRootID,
			ReplyCount:   m.ReplyCount,
			CreatedAt:    m.CreatedAt.Time,
			UpdatedAt:    m.UpdatedAt.Time,
		}
	}

	return result, nil
}

// DeleteChannelMessage removes a message and its replies, returning NotFoundError if absent.
func (s *MessagingService) DeleteChannelMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := s.database.DeleteChannelMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("delete channel message: %w", err)
	}
	if rows == 0 {
		return NotFoundError{What: "channel message"}
	}

	return nil
}
