package project

import "errors"

var (
	ErrEmptyProjectID        = errors.New("project id cannot be empty")
	ErrEmptyProjectName      = errors.New("project name cannot be empty")
	ErrEmptyProjectUpdatedAt = errors.New("project updatedAt cannot be empty")
)
