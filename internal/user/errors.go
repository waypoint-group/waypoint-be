package user

import "errors"

var (
	ErrNotFound           = errors.New("user not found")
	ErrUserNameTaken      = errors.New("user name already in use")
	ErrEmailTaken         = errors.New("email already in use")
	ErrIdentityTaken      = errors.New("user identity already in use")
	ErrInvalidId          = errors.New("invalid user id")
	ErrInvalidUserName    = errors.New("invalid user name")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidDisplayName = errors.New("invalid display name")
	ErrInvalidIdentity    = errors.New("identity issuer and subject are required")
)
