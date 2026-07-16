package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/user"
)

type session struct {
	userID uuid.UUID
	role   user.Role
}

type MemoryTokenService struct {
	mu       sync.RWMutex
	sessions map[string]session
}

func NewMemoryTokenService() *MemoryTokenService {
	return &MemoryTokenService{
		sessions: make(map[string]session),
	}
}

func (s *MemoryTokenService) Generate(userID uuid.UUID, role user.Role) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	token := hex.EncodeToString(bytes)

	s.sessions[token] = session{
		userID: userID,
		role:   role,
	}

	return token, nil
}

func (s *MemoryTokenService) Validate(token string) (uuid.UUID, user.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[token]
	if !ok {
		return uuid.Nil, user.RoleGhost, errors.New("invalid or expired token")
	}

	return sess.userID, sess.role, nil
}
