package auth

import (
	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
	"github.com/safarislava/typstlab-server/internal/domain/user"
)

type TokenService interface {
	Generate(userID uuid.UUID, role user.Role) (token.Token, error)
	Validate(t token.Token) (uuid.UUID, user.Role, error)
}
