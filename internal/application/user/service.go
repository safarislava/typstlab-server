package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

type RegisterRequest struct {
	Email    string
	Password string
	Role     string
}

type RegisterResponse struct {
	ID    uuid.UUID
	Email string
	Role  domain.Role
}

type Service struct {
	repo   Repository
	hasher PasswordHasher
}

func NewService(repo Repository, hasher PasswordHasher) *Service {
	return &Service{
		repo:   repo,
		hasher: hasher,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	existing, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, errors.New("user with this email already exists")
	}

	hashedPassword, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	role := domain.RoleUser
	if req.Role != "" {
		role = domain.Role(req.Role)
	}

	u, err := domain.NewUser(uuid.New(), req.Email, hashedPassword, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.repo.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return &RegisterResponse{
		ID:    u.ID(),
		Email: u.Email(),
		Role:  u.Role(),
	}, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return u, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return u, nil
}
