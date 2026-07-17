package session

import (
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
)

func NewSession(t token.Token, userID uuid.UUID, expiresAt time.Time) (Session, error) {
	if t.Value() == "" {
		return Session{}, token.ErrInvalidTokenValue
	}
	if userID == uuid.Nil {
		return Session{}, ErrInvalidUserID
	}
	if expiresAt.Before(time.Now()) {
		return Session{}, ErrExpiredSession
	}
	return Session{
		token:     t,
		userID:    userID,
		expiresAt: expiresAt,
	}, nil
}
