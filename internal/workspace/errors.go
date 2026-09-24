package workspace

import "errors"

var (
	ErrNotFound          = errors.New("workspace not found")
	ErrMemberNotFound    = errors.New("workspace member not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrForbidden         = errors.New("insufficient workspace permissions")
	ErrInvalidID         = errors.New("invalid workspace or user id")
	ErrInvalidName       = errors.New("workspace name cannot be empty")
	ErrAlreadyMember     = errors.New("user is already a workspace member")
	ErrOwnerRemoval      = errors.New("cannot remove workspace owner")
	ErrWorkspaceNotEmpty = errors.New("remove other workspace members before deleting the workspace")
)
