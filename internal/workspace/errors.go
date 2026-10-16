package workspace

import "errors"

var (
	ErrForbidden               = errors.New("insufficient workspace permissions")
	ErrWorkspaceNotFound       = errors.New("workspace not found")
	ErrWorkspaceMemberNotFound = errors.New("workspace member not found")
	ErrUserNotFound            = errors.New("user not found")
	ErrInvalidID               = errors.New("invalid workspace or user id")
	ErrInvalidName             = errors.New("workspace name cannot be empty")
	ErrAlreadyMember           = errors.New("user is already a workspace member")
	ErrOwnerRemoval            = errors.New("cannot remove workspace owner")
)
