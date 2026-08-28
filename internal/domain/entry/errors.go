package entry

import "errors"

var (
	ErrEmptyID   = errors.New("entry id cannot be empty")
	ErrEmptyName = errors.New("entry name cannot be empty")
)
