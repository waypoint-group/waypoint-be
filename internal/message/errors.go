package message

import "errors"

var (
	ErrNotFound             = errors.New("message not found")
	ErrEmptyBody            = errors.New("message body cannot be empty")
	ErrMessageToSelf        = errors.New("author and recipient cannot be the same")
	ErrInvalidLimit         = errors.New("limit exceeds maximum allowed value")
	ErrInvalidOffset        = errors.New("offset exceeds maximum allowed value")
	ErrConversationWithSelf = errors.New("conversation must be between distinct users")
)
