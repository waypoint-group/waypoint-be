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

// DirectMessageStore defines the data access interface for direct
// message operations
type DirectMessageStore interface {
	CreateDirectMessage(context.Context, sqlc.CreateDirectMessageParams) (sqlc.DirectMessage, error)
	ListDirectMessages(context.Context, sqlc.ListDirectMessagesParams) ([]sqlc.DirectMessage, error)
	SelectDirectMessage(context.Context, uuid.UUID) (sqlc.DirectMessage, error)
	UpdateDirectMessageBody(context.Context, sqlc.UpdateDirectMessageBodyParams) (sqlc.DirectMessage, error)
	DeleteDirectMessage(context.Context, uuid.UUID) (int64, error)
}

// DirectMessageService provides business operations for direct
// messages.
type DirectMessageService struct {
	store DirectMessageStore
}

// DirectMessage is a message sent to a direct in the Waypoint domain.
type DirectMessage struct {
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

func NewDirectMessageService(store DirectMessageStore) *DirectMessageService {
	return &DirectMessageService{
		store: store,
	}
}

// CreateDirectMessage creates a new direct message and returns it.
func (s *DirectMessageService) CreateDirectMessage(ctx context.Context,
	authorID uuid.UUID,
	recipientID uuid.UUID,
	body string,
) (*DirectMessage, error) {
	directMsg, err := s.store.CreateDirectMessage(ctx, sqlc.CreateDirectMessageParams{
		ID:          uuid.NewV7(),
		AuthorID:    authorID,
		RecipientID: recipientID,
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("create direct message: %w", err)
	}

	return &DirectMessage{
		ID:          directMsg.ID,
		AuthorID:    directMsg.AuthorID,
		RecipientID: directMsg.RecipientID,
		Body:        directMsg.Body,
		CreatedAt:   directMsg.CreatedAt.Time,
		UpdatedAt:   directMsg.UpdatedAt.Time,
	}, nil
}

// GetDirectMessage retrieves a direct message by its ID.
func (s *DirectMessageService) GetDirectMessage(ctx context.Context, id uuid.UUID) (*DirectMessage, error) {
	directMsg, err := s.store.SelectDirectMessage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{
				What: "direct message",
			}
		}

		return nil, fmt.Errorf("get direct message: %w", err)
	}

	return &DirectMessage{
		ID:          directMsg.ID,
		AuthorID:    directMsg.AuthorID,
		RecipientID: directMsg.RecipientID,
		Body:        directMsg.Body,
		CreatedAt:   directMsg.CreatedAt.Time,
		UpdatedAt:   directMsg.UpdatedAt.Time,
	}, nil
}

// GetDirectMessagePage returns a page of direct messages for a given
// (author ID, recipient ID) pair.
func (s *DirectMessageService) GetDirectMessagePage(ctx context.Context,
	authorID uuid.UUID,
	recipientID uuid.UUID,
	limit int32, offset int32,
) ([]DirectMessage, error) {
	directMsgs, err := s.store.ListDirectMessages(ctx, sqlc.ListDirectMessagesParams{
		AuthorID:   authorID,
		AuthorID_2: recipientID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get direct message page: %w", err)
	}

	result := make([]DirectMessage, 0, len(directMsgs))
	for _, msg := range directMsgs {
		result = append(result, DirectMessage{
			ID:          msg.ID,
			AuthorID:    msg.AuthorID,
			RecipientID: msg.RecipientID,
			Body:        msg.Body,
			CreatedAt:   msg.CreatedAt.Time,
			UpdatedAt:   msg.UpdatedAt.Time,
		})
	}

	return result, nil
}

// UpdateDirectMessage updates the body of an existing direct message.
func (s *DirectMessageService) UpdateDirectMessage(ctx context.Context,
	id uuid.UUID,
	body string,
) (*DirectMessage, error) {
	directMsg, err := s.store.UpdateDirectMessageBody(ctx, sqlc.UpdateDirectMessageBodyParams{
		ID:   id,
		Body: body,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "direct message"}
		}
		return nil, fmt.Errorf("update direct message: %w", err)
	}

	return &DirectMessage{
		ID:          directMsg.ID,
		AuthorID:    directMsg.AuthorID,
		RecipientID: directMsg.RecipientID,
		Body:        directMsg.Body,
		CreatedAt:   directMsg.CreatedAt.Time,
		UpdatedAt:   directMsg.UpdatedAt.Time,
	}, nil
}

// DeleteDirectMessage deletes a direct message by its ID.
func (s *DirectMessageService) DeleteDirectMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := s.store.DeleteDirectMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("delete direct message: %w", err)
	}
	if rows == 0 {
		return NotFoundError{What: "direct message"}
	}

	return nil
}
