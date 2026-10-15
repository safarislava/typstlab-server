package invite

import (
	"time"

	"github.com/google/uuid"
)

func NewInvite(id, projectID, createdBy uuid.UUID, code string, expiresAt, createdAt time.Time) (*Invite, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyInviteID
	}
	if projectID == uuid.Nil {
		return nil, ErrEmptyProjectID
	}
	if createdBy == uuid.Nil {
		return nil, ErrEmptyCreatedBy
	}
	if code == "" {
		return nil, ErrEmptyInviteCode
	}
	if expiresAt.IsZero() {
		return nil, ErrEmptyExpiresAt
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return &Invite{
		id:        id,
		projectID: projectID,
		code:      code,
		createdBy: createdBy,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}, nil
}
