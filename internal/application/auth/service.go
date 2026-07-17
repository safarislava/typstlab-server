package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	userApp "github.com/safarislava/typstlab-server/internal/application/user"
	sessionDomain "github.com/safarislava/typstlab-server/internal/domain/session"
	tokenDomain "github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

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

type UseCase interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Refresh(ctx context.Context, req RefreshRequest) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken tokenDomain.Token) error
	Authorize(t tokenDomain.Token) (uuid.UUID, domainUser.Role, error)
}

type Service struct {
	userService         userApp.UseCase
	refreshTokenService sessionApp.UseCase
	tokenService        TokenService
	hasher              PasswordHasher
}

func NewService(
	userService userApp.UseCase,
	refreshTokenService sessionApp.UseCase,
	tokenService TokenService,
	hasher PasswordHasher,
) UseCase {
	return &Service{
		userService:         userService,
		refreshTokenService: refreshTokenService,
		tokenService:        tokenService,
		hasher:              hasher,
	}
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.userService.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	err = s.hasher.Compare(u.PasswordHash(), req.Password)
	if err != nil {
		return nil, errors.New("invalid email or password")
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
		return nil, errors.New("invalid or expired refresh token")
	}

	if session.IsExpired() {
		_ = s.refreshTokenService.Invalidate(ctx, req.RefreshToken)
		return nil, errors.New("invalid or expired refresh token")
	}

	u, err := s.userService.GetByID(ctx, session.UserID())
	if err != nil {
		return nil, errors.New("user not found")
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
