package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

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

func (s *JWTTokenService) Generate(userID uuid.UUID, role user.Role) (string, error) {
	expirationTime := time.Now().Add(s.tokenDuration)

	c := claims{
		Role: string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	tokenStr, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenStr, nil
}

func (s *JWTTokenService) Validate(tokenStr string) (uuid.UUID, user.Role, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return uuid.Nil, user.RoleGhost, fmt.Errorf("token validation failed: %w", err)
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return uuid.Nil, user.RoleGhost, errors.New("invalid token claims")
	}

	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, user.RoleGhost, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return userID, user.Role(c.Role), nil
}
