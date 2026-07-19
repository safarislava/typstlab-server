package block

import "errors"

var (
	ErrEmptyBlockID    = errors.New("block id cannot be empty")
	ErrEmptyBlockState = errors.New("block state cannot be empty")
)
