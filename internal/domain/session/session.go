package session

import (
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
)

type Session struct {
	token     token.Token
	userID    uuid.UUID
	expiresAt time.Time
}

func (s Session) Token() token.Token {
	return s.token
}

func (s Session) UserID() uuid.UUID {
	return s.userID
}

func (s Session) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}
