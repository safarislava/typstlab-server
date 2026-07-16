package user

import (
	"strings"

	"github.com/google/uuid"
)

func NewUser(id uuid.UUID, email, passwordHash string, role Role) (*User, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if !strings.Contains(email, "@") || len(email) < 3 {
		return nil, ErrInvalidEmail
	}
	if passwordHash == "" {
		return nil, ErrEmptyPasswordHash
	}
	if role != RoleUser && role != RoleAdmin {
		return nil, ErrInvalidRole
	}

	return &User{
		id:           id,
		email:        strings.ToLower(strings.TrimSpace(email)),
		passwordHash: passwordHash,
		role:         role,
	}, nil
}
