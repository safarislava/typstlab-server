package session

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/session"
	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
)

type Repository interface {
	Save(ctx context.Context, s domain.Session) error
	FindByToken(ctx context.Context, t domainToken.Token) (domain.Session, error)
	Delete(ctx context.Context, t domainToken.Token) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
