package file

import "errors"

var (
	ErrEmptyFileID    = errors.New("file id cannot be empty")
	ErrEmptyProjectID = errors.New("project id cannot be empty")
	ErrEmptyFileName  = errors.New("file name cannot be empty")
)
