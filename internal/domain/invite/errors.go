package invite

import "errors"

var (
	ErrEmptyInviteID   = errors.New("invite id cannot be empty")
	ErrEmptyProjectID  = errors.New("project id cannot be empty")
	ErrEmptyCreatedBy  = errors.New("created by user id cannot be empty")
	ErrEmptyInviteCode = errors.New("invite code cannot be empty")
	ErrEmptyExpiresAt  = errors.New("invite expires at cannot be empty")
	ErrInviteNotFound  = errors.New("invite not found")
	ErrInviteExpired   = errors.New("invite has expired")
)
