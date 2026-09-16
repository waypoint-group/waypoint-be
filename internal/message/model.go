package message

import (
	"time"
	"uuid"
)

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
