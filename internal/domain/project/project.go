package project

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	id        uuid.UUID
	userIDs   []uuid.UUID
	name      string
	updatedAt time.Time
}

func (p *Project) ID() uuid.UUID {
	return p.id
}

func (p *Project) UserIDs() []uuid.UUID {
	return append([]uuid.UUID(nil), p.userIDs...)
}

func (p *Project) HasUser(userID uuid.UUID) bool {
	for _, id := range p.userIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (p *Project) AddUser(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return ErrEmptyUserID
	}
	if p.HasUser(userID) {
		return nil
	}
	p.userIDs = append(p.userIDs, userID)
	return nil
}

func (p *Project) Name() string {
	return p.name
}

func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}
