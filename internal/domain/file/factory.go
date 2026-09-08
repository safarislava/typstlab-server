package file

import (
	"time"

	"github.com/google/uuid"
)

func NewBinaryFile(id, projectID uuid.UUID, name string, content []byte, updatedAt time.Time) (*BinaryFile, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyFileID
	}
	if projectID == uuid.Nil {
		return nil, ErrEmptyProjectID
	}
	if name == "" {
		return nil, ErrEmptyFileName
	}
	return &BinaryFile{
		id:        id,
		projectID: projectID,
		name:      name,
		content:   append([]byte(nil), content...),
		updatedAt: updatedAt,
	}, nil
}

func NewTypstFile(id, projectID uuid.UUID, name string, state []byte, updatedAt time.Time) (*TypstFile, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyFileID
	}
	if projectID == uuid.Nil {
		return nil, ErrEmptyProjectID
	}
	if name == "" {
		return nil, ErrEmptyFileName
	}
	return &TypstFile{
		id:        id,
		projectID: projectID,
		name:      name,
		state:     append([]byte(nil), state...),
		updatedAt: updatedAt,
	}, nil
}
