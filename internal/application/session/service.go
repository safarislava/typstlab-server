package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/session"
	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
)

type UseCase interface {
	Create(ctx context.Context, userID uuid.UUID, duration time.Duration) (domain.Session, error)
	Get(ctx context.Context, token domainToken.Token) (domain.Session, error)
	Invalidate(ctx context.Context, token domainToken.Token) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) UseCase {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, duration time.Duration) (domain.Session, error) {
	raw, err := generateRandomHex(32)
	if err != nil {
		return domain.Session{}, err
	}

	rt, err := domainToken.NewToken(raw)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to create token value: %w", err)
	}

	session, err := domain.NewSession(rt, userID, time.Now().Add(duration))
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to construct session: %w", err)
	}

	if err := s.repo.Save(ctx, session); err != nil {
		return domain.Session{}, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (s *Service) Get(ctx context.Context, token domainToken.Token) (domain.Session, error) {
	rt, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to find session: %w", err)
	}
	return rt, nil
}

func (s *Service) Invalidate(ctx context.Context, token domainToken.Token) error {
	if err := s.repo.Delete(ctx, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func generateRandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
