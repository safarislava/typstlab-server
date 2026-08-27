package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	sessionDomain "github.com/safarislava/typstlab-server/internal/domain/session"
	tokenDomain "github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type UserService interface {
	GetByEmail(ctx context.Context, email string) (*domainUser.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

type SessionService interface {
	Create(ctx context.Context, userID uuid.UUID, duration time.Duration) (sessionDomain.Session, error)
	Get(ctx context.Context, token tokenDomain.Token) (sessionDomain.Session, error)
	Invalidate(ctx context.Context, token tokenDomain.Token) error
}

type TokenService interface {
	Generate(userID uuid.UUID, role domainUser.Role) (tokenDomain.Token, error)
	Validate(t tokenDomain.Token) (uuid.UUID, domainUser.Role, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	AccessToken  tokenDomain.Token
	RefreshToken sessionDomain.Session
}

type RefreshRequest struct {
	RefreshToken tokenDomain.Token
}

type RefreshResponse struct {
	AccessToken  tokenDomain.Token
	RefreshToken sessionDomain.Session
}

type Service struct {
	userService         UserService
	refreshTokenService SessionService
	tokenService        TokenService
	hasher              PasswordHasher
}

func NewService(
	userService UserService,
	refreshTokenService SessionService,
	tokenService TokenService,
	hasher PasswordHasher,
) *Service {
	return &Service{
		userService:         userService,
		refreshTokenService: refreshTokenService,
		tokenService:        tokenService,
		hasher:              hasher,
	}
}

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.userService.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	err = s.hasher.Compare(u.PasswordHash(), req.Password)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	t, err := s.tokenService.Generate(u.ID(), u.Role())
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	rt, err := s.refreshTokenService.Create(ctx, u.ID(), 30*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  t,
		RefreshToken: rt,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (*RefreshResponse, error) {
	session, err := s.refreshTokenService.Get(ctx, req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if session.IsExpired() {
		_ = s.refreshTokenService.Invalidate(ctx, req.RefreshToken)
		return nil, ErrInvalidRefreshToken
	}

	u, err := s.userService.GetByID(ctx, session.UserID())
	if err != nil {
		return nil, domainUser.ErrUserNotFound
	}

	t, err := s.tokenService.Generate(u.ID(), u.Role())
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	rt, err := s.refreshTokenService.Create(ctx, u.ID(), 30*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}

	_ = s.refreshTokenService.Invalidate(ctx, req.RefreshToken)

	return &RefreshResponse{
		AccessToken:  t,
		RefreshToken: rt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken tokenDomain.Token) error {
	if err := s.refreshTokenService.Invalidate(ctx, refreshToken); err != nil {
		return fmt.Errorf("failed to invalidate refresh token: %w", err)
	}
	return nil
}

func (s *Service) Authorize(t tokenDomain.Token) (uuid.UUID, domainUser.Role, error) {
	userID, role, err := s.tokenService.Validate(t)
	if err != nil {
		return uuid.Nil, domainUser.RoleGhost, fmt.Errorf("failed to validate token: %w", err)
	}
	return userID, role, nil
}
