package project

import (
	"time"

	"github.com/google/uuid"
)

func NewProject(id uuid.UUID, name string, updatedAt time.Time) (*Project, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyProjectID
	}
	if name == "" {
		return nil, ErrEmptyProjectName
	}
	if updatedAt.IsZero() {
		return nil, ErrEmptyProjectUpdatedAt
	}

	return &Project{
		id:        id,
		name:      name,
		updatedAt: updatedAt,
	}, nil
}
