package project

import "errors"

var (
	ErrEmptyID        = errors.New("project id cannot be empty")
	ErrEmptyName      = errors.New("project name cannot be empty")
	ErrEmptyUpdatedAt = errors.New("project updatedAt cannot be empty")
)
