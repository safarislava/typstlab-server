package invite

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	id        uuid.UUID
	projectID uuid.UUID
	code      string
	createdBy uuid.UUID
	expiresAt time.Time
	createdAt time.Time
}

func (i *Invite) ID() uuid.UUID {
	return i.id
}

func (i *Invite) ProjectID() uuid.UUID {
	return i.projectID
}

func (i *Invite) Code() string {
	return i.code
}

func (i *Invite) CreatedBy() uuid.UUID {
	return i.createdBy
}

func (i *Invite) ExpiresAt() time.Time {
	return i.expiresAt
}

func (i *Invite) CreatedAt() time.Time {
	return i.createdAt
}

func (i *Invite) IsExpired(now time.Time) bool {
	return !i.expiresAt.IsZero() && now.After(i.expiresAt)
}
