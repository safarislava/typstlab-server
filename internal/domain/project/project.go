package project

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	id        uuid.UUID
	name      string
	updatedAt time.Time
}

func (p *Project) ID() uuid.UUID {
	return p.id
}

func (p *Project) Name() string {
	return p.name
}

func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}
