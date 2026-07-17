package session

import "errors"

var (
	ErrInvalidUserID   = errors.New("user ID cannot be nil or empty")
	ErrExpiredSession  = errors.New("session has expired")
	ErrSessionNotFound = errors.New("session not found")
)
