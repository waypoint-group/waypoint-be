package message

import (
	"context"
	"errors"
	"fmt"
	"math"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// Service manages direct messages and channel messages.
type Service struct {
	database *db.Database
}

func NewService(database *db.Database) *Service {
	return &Service{database: database}
}

// SendDirectMessage stores a nonempty message between distinct users.
func (s *Service) SendDirectMessage(ctx context.Context, dm DirectMessage) (*SentDirectMessage, error) {
	if dm.Body == "" {
		return nil, ErrEmptyBody
	}
	if dm.AuthorID == dm.RecipientID {
		return nil, ErrMessageToSelf
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
func (s *Service) UpdateDirectMessage(ctx context.Context, id uuid.UUID, newBody string) (*SentDirectMessage, error) {
	if newBody == "" {
		return nil, ErrEmptyBody
	}

	updated, err := s.database.UpdateDirectMessageBody(ctx, sqlc.UpdateDirectMessageBodyParams{
		ID:   id,
		Body: newBody,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("direct message: %w", ErrNotFound)
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
func (s *Service) ReadSingleDirectMessage(
	ctx context.Context,
	id uuid.UUID,
) (*SentDirectMessage, error) {
	msg, err := s.database.SelectDirectMessage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("direct message: %w", ErrNotFound)
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
func (s *Service) ReadDirectMessagePage(
	ctx context.Context,
	userA uuid.UUID,
	userB uuid.UUID,
	limit uint32,
	offset uint32,
) ([]SentDirectMessage, error) {
	if limit > math.MaxInt32 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLimit, limit)
	}
	if offset > math.MaxInt32 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidOffset, offset)
	}
	if userA == userB {
		return nil, ErrConversationWithSelf
	}

	messages, err := s.database.ListDirectMessagesBetween(ctx, sqlc.ListDirectMessagesBetweenParams{
		UserA:  userA,
		UserB:  userB,
		Limit:  int32(limit),
		Offset: int32(offset),
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

// DeleteDirectMessage removes a message, returning ErrNotFound if absent.
func (s *Service) DeleteDirectMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := s.database.DeleteDirectMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("delete direct message: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("direct message: %w", ErrNotFound)
	}

	return nil
}

// SendChannelMessage stores a nonempty channel message or thread reply.
func (s *Service) SendChannelMessage(ctx context.Context, cm ChannelMessage) (*SentChannelMessage, error) {
	if cm.Body == "" {
		return nil, ErrEmptyBody
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
func (s *Service) UpdateChannelMessage(ctx context.Context, id uuid.UUID, newBody string) (*SentChannelMessage, error) {
	if newBody == "" {
		return nil, ErrEmptyBody
	}

	updated, err := s.database.UpdateChannelMessageBody(ctx, sqlc.UpdateChannelMessageBodyParams{
		ID:   id,
		Body: newBody,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("channel message: %w", ErrNotFound)
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
func (s *Service) ReadSingleChannelMessage(
	ctx context.Context,
	id uuid.UUID,
) (*SentChannelMessage, error) {
	msg, err := s.database.SelectChannelMessage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("channel message: %w", ErrNotFound)
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
func (s *Service) ReadChannelMessagePage(
	ctx context.Context,
	channelID uuid.UUID,
	limit uint32,
	offset uint32,
) ([]SentChannelMessage, error) {
	if limit > math.MaxInt32 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLimit, limit)
	}
	if offset > math.MaxInt32 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidOffset, offset)
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

// ReadChannelThreadReplies returns replies ordered by creation time.
func (s *Service) ReadChannelThreadReplies(ctx context.Context, threadRootID uuid.UUID) ([]SentChannelMessage, error) {
	messages, err := s.database.ListChannelThreadReplies(ctx, &threadRootID)
	if err != nil {
		return nil, fmt.Errorf("read channel thread replies: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("channel thread replies: %w", ErrNotFound)
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

// DeleteChannelMessage removes a message and its replies, returning ErrNotFound if absent.
func (s *Service) DeleteChannelMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := s.database.DeleteChannelMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("delete channel message: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("channel message: %w", ErrNotFound)
	}

	return nil
}
