package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
	"github.com/safarislava/typstlab-server/internal/domain/user"
)

type JWTTokenService struct {
	secretKey     []byte
	tokenDuration time.Duration
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTTokenService(secretKey string, duration time.Duration) *JWTTokenService {
	return &JWTTokenService{
		secretKey:     []byte(secretKey),
		tokenDuration: duration,
	}
}

func (s *JWTTokenService) Generate(userID uuid.UUID, role user.Role) (token.Token, error) {
	expirationTime := time.Now().Add(s.tokenDuration)

	c := claims{
		Role: string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	tokenStr, err := jwtToken.SignedString(s.secretKey)
	if err != nil {
		return token.Token{}, fmt.Errorf("failed to sign token: %w", err)
	}

	t, err := token.NewToken(tokenStr)
	if err != nil {
		return token.Token{}, fmt.Errorf("failed to create domain token: %w", err)
	}
	return t, nil
}

func (s *JWTTokenService) Validate(t token.Token) (uuid.UUID, user.Role, error) {
	parsedToken, err := jwt.ParseWithClaims(t.Value(), &claims{}, func(jwtTok *jwt.Token) (interface{}, error) {
		if _, ok := jwtTok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", jwtTok.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return uuid.Nil, user.RoleGhost, fmt.Errorf("token validation failed: %w", err)
	}

	c, ok := parsedToken.Claims.(*claims)
	if !ok || !parsedToken.Valid {
		return uuid.Nil, user.RoleGhost, errors.New("invalid token claims")
	}

	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, user.RoleGhost, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return userID, user.Role(c.Role), nil
}
