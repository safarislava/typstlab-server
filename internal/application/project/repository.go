package project

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type Repository interface {
	Save(ctx context.Context, p *domain.Project) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error)
}
