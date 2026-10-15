package project

import "errors"

var (
	ErrEmptyID           = errors.New("project id cannot be empty")
	ErrEmptyName         = errors.New("project name cannot be empty")
	ErrEmptyUpdatedAt    = errors.New("project updatedAt cannot be empty")
	ErrEmptyUserID       = errors.New("project user id cannot be empty")
	ErrNoUsers           = errors.New("project must have at least one user")
	ErrProjectNotFound   = errors.New("project not found")
	ErrUserNotInProject  = errors.New("user is not a member of the project")
	ErrUserAlreadyMember = errors.New("user is already a member of the project")
)
