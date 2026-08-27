package file

import "errors"

var (
	ErrEmptyFileID        = errors.New("file id cannot be empty")
	ErrEmptyProjectID     = errors.New("project id cannot be empty")
	ErrEmptyFileName      = errors.New("file name cannot be empty")
	ErrNilState           = errors.New("state cannot be nil")
	ErrFileNotFound       = errors.New("file not found")
	ErrTypstFileNotFound  = errors.New("typst file not found")
	ErrBinaryFileNotFound = errors.New("binary file not found")
)
