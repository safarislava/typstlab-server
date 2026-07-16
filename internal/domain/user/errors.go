package user

import "errors"

var (
	ErrEmptyID           = errors.New("user id cannot be empty")
	ErrInvalidEmail      = errors.New("invalid user email")
	ErrEmptyPasswordHash = errors.New("password hash cannot be empty")
	ErrInvalidRole       = errors.New("invalid user role")
)
