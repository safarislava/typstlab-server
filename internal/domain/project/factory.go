package project

import (
	"time"

	"github.com/google/uuid"
)

func NewProject(id uuid.UUID, userIDs []uuid.UUID, name string, updatedAt time.Time) (*Project, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if len(userIDs) == 0 {
		return nil, ErrNoUsers
	}
	for _, userID := range userIDs {
		if userID == uuid.Nil {
			return nil, ErrEmptyUserID
		}
	}
	if name == "" {
		return nil, ErrEmptyName
	}
	if updatedAt.IsZero() {
		return nil, ErrEmptyUpdatedAt
	}

	return &Project{
		id:        id,
		userIDs:   append([]uuid.UUID(nil), userIDs...),
		name:      name,
		updatedAt: updatedAt,
	}, nil
}
