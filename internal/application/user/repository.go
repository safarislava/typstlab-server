package user

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

type Repository interface {
	Save(ctx context.Context, u *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}
