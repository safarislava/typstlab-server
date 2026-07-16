package project

import (
	"time"

	"github.com/google/uuid"
)

func NewProject(id uuid.UUID, name string, updatedAt time.Time) (*Project, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if name == "" {
		return nil, ErrEmptyName
	}
	if updatedAt.IsZero() {
		return nil, ErrEmptyUpdatedAt
	}

	return &Project{
		id:        id,
		name:      name,
		updatedAt: updatedAt,
	}, nil
}
