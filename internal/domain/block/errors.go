package block

import "errors"

var (
	ErrEmptyBlockID   = errors.New("file id cannot be empty")
	ErrEmptyBlockCrdt = errors.New("project id cannot be empty")
)
