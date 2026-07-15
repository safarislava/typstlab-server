package project

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type CreateProjectRequest struct {
	Name string
}

type CreateProjectResponse struct {
	ID        uuid.UUID
	Name      string
	UpdatedAt time.Time
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateProject(ctx context.Context, request CreateProjectRequest) (*CreateProjectResponse, error) {
	p, err := domain.NewProject(uuid.New(), request.Name, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create domain project: %w", err)
	}

	if err := s.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return &CreateProjectResponse{
		ID:        p.ID(),
		Name:      p.Name(),
		UpdatedAt: p.UpdatedAt(),
	}, nil
}
