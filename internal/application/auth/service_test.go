package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	userApp "github.com/safarislava/typstlab-server/internal/application/user"
	sessionDomain "github.com/safarislava/typstlab-server/internal/domain/session"
	tokenDomain "github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

const (
	testLoginEmail   = "login@example.com"
	testRefreshToken = "valid_refresh_token"
	testPassword     = "password"
)

type mockUserRepo struct {
	saveFunc        func(ctx context.Context, u *domainUser.User) error
	findByEmailFunc func(ctx context.Context, email string) (*domainUser.User, error)
	findByIDFunc    func(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

func (m *mockUserRepo) Save(ctx context.Context, u *domainUser.User) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domainUser.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(ctx, email)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

type mockHasher struct {
	hashFunc    func(password string) (string, error)
	compareFunc func(hashedPassword, password string) error
}

func (m *mockHasher) Hash(password string) (string, error) {
	if m.hashFunc != nil {
		return m.hashFunc(password)
	}
	return "hashed_" + password, nil
}

func (m *mockHasher) Compare(hashedPassword, password string) error {
	if m.compareFunc != nil {
		return m.compareFunc(hashedPassword, password)
	}
	if hashedPassword == "hashed_"+password {
		return nil
	}
	return errors.New("mismatch")
}

type mockTokenService struct {
	generateFunc func(userID uuid.UUID, role domainUser.Role) (tokenDomain.Token, error)
	validateFunc func(t tokenDomain.Token) (uuid.UUID, domainUser.Role, error)
}

func (m *mockTokenService) Generate(userID uuid.UUID, role domainUser.Role) (tokenDomain.Token, error) {
	if m.generateFunc != nil {
		return m.generateFunc(userID, role)
	}
	t, err := tokenDomain.NewToken("token_" + userID.String())
	if err != nil {
		return tokenDomain.Token{}, fmt.Errorf("failed to create mock token: %w", err)
	}
	return t, nil
}

func (m *mockTokenService) Validate(t tokenDomain.Token) (uuid.UUID, domainUser.Role, error) {
	if m.validateFunc != nil {
		return m.validateFunc(t)
	}
	return uuid.Nil, domainUser.RoleGhost, errors.New("invalid")
}

type mockRefreshTokenRepo struct {
	saveFunc           func(ctx context.Context, rt sessionDomain.Session) error
	findByTokenFunc    func(ctx context.Context, t tokenDomain.Token) (sessionDomain.Session, error)
	deleteFunc         func(ctx context.Context, t tokenDomain.Token) error
	deleteByUserIDFunc func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockRefreshTokenRepo) Save(ctx context.Context, rt sessionDomain.Session) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, rt)
	}
	return nil
}

func (m *mockRefreshTokenRepo) FindByToken(ctx context.Context, t tokenDomain.Token) (sessionDomain.Session, error) {
	if m.findByTokenFunc != nil {
		return m.findByTokenFunc(ctx, t)
	}
	return sessionDomain.Session{}, sessionDomain.ErrSessionNotFound
}

func (m *mockRefreshTokenRepo) Delete(ctx context.Context, t tokenDomain.Token) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, t)
	}
	return nil
}

func (m *mockRefreshTokenRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFunc != nil {
		return m.deleteByUserIDFunc(ctx, userID)
	}
	return nil
}

func TestService_Login_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	u, _ := domainUser.NewUser(userID, testLoginEmail, "hashed_secret", domainUser.RoleUser)
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	req := LoginRequest{
		Email:    testLoginEmail,
		Password: "secret",
	}

	resp, err := svc.Login(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedToken := "token_" + userID.String()
	if resp.AccessToken.Value() != expectedToken {
		t.Errorf("Expected token %q, got %q", expectedToken, resp.AccessToken.Value())
	}
	if resp.RefreshToken.Token().Value() == "" {
		t.Error("Expected non-empty refresh token")
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			return nil, errors.New("not found")
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	req := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: testPassword,
	}

	_, err := svc.Login(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error user not found, got nil")
	}
}

func TestService_Login_PasswordMismatch(t *testing.T) {
	t.Parallel()

	u, _ := domainUser.NewUser(uuid.New(), testLoginEmail, "hashed_secret", domainUser.RoleUser)
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{
		compareFunc: func(hashedPassword, password string) error {
			return errors.New("mismatch")
		},
	}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	req := LoginRequest{
		Email:    testLoginEmail,
		Password: "wrongpassword",
	}

	_, err := svc.Login(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error password mismatch, got nil")
	}
}

func TestService_Login_TokenGenerationFailure(t *testing.T) {
	t.Parallel()

	u, _ := domainUser.NewUser(uuid.New(), testLoginEmail, "hashed_secret", domainUser.RoleUser)
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{
		generateFunc: func(userID uuid.UUID, role domainUser.Role) (tokenDomain.Token, error) {
			return tokenDomain.Token{}, errors.New("failed to generate")
		},
	}
	rtRepo := &mockRefreshTokenRepo{}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	req := LoginRequest{
		Email:    testLoginEmail,
		Password: "password",
	}

	_, err := svc.Login(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error token generation failure, got nil")
	}
}

func TestService_Refresh_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	u, _ := domainUser.NewUser(userID, testLoginEmail, "hashed_secret", domainUser.RoleUser)
	rtVal, _ := tokenDomain.NewToken(testRefreshToken)
	rt, _ := sessionDomain.NewSession(rtVal, userID, time.Now().Add(1*time.Hour))

	userRepo := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{
		findByTokenFunc: func(ctx context.Context, t tokenDomain.Token) (sessionDomain.Session, error) {
			return rt, nil
		},
	}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	rtToken, _ := tokenDomain.NewToken(testRefreshToken)
	resp, err := svc.Refresh(context.Background(), RefreshRequest{RefreshToken: rtToken})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedAccessToken := "token_" + userID.String()
	if resp.AccessToken.Value() != expectedAccessToken {
		t.Errorf("Expected access token %q, got %q", expectedAccessToken, resp.AccessToken.Value())
	}
	if resp.RefreshToken.Token().Value() == "" {
		t.Error("Expected rotated refresh token to be non-empty")
	}
	if resp.RefreshToken.Token().Value() == testRefreshToken {
		t.Error("Expected refresh token to be rotated, but got the old one")
	}
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepo{}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{
		findByTokenFunc: func(ctx context.Context, t tokenDomain.Token) (sessionDomain.Session, error) {
			return sessionDomain.Session{}, sessionDomain.ErrSessionNotFound
		},
	}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	invalidToken, _ := tokenDomain.NewToken("invalid")
	_, err := svc.Refresh(context.Background(), RefreshRequest{RefreshToken: invalidToken})
	if err == nil {
		t.Fatal("Expected error for invalid refresh token, got nil")
	}
}

func TestService_Refresh_ExpiredToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	rtVal, _ := tokenDomain.NewToken("expired_token")
	rt, _ := sessionDomain.NewSession(rtVal, userID, time.Now().Add(-1*time.Hour))

	userRepo := &mockUserRepo{}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{
		findByTokenFunc: func(ctx context.Context, t tokenDomain.Token) (sessionDomain.Session, error) {
			return rt, nil
		},
	}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	expiredToken, _ := tokenDomain.NewToken("expired_token")
	_, err := svc.Refresh(context.Background(), RefreshRequest{RefreshToken: expiredToken})
	if err == nil {
		t.Fatal("Expected error for expired refresh token, got nil")
	}
}

func TestService_Logout_Success(t *testing.T) {
	t.Parallel()

	deleted := false
	userRepo := &mockUserRepo{}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}
	rtRepo := &mockRefreshTokenRepo{
		deleteFunc: func(ctx context.Context, t tokenDomain.Token) error {
			if t.Value() == "logout_token" {
				deleted = true
			}
			return nil
		},
	}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	logoutToken, _ := tokenDomain.NewToken("logout_token")
	err := svc.Logout(context.Background(), logoutToken)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleted {
		t.Error("Expected refresh token to be deleted from repo")
	}
}

func TestService_ValidateAccessToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	role := domainUser.RoleUser
	tok, _ := tokenDomain.NewToken("valid_access_token")

	userRepo := &mockUserRepo{}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{
		validateFunc: func(tk tokenDomain.Token) (uuid.UUID, domainUser.Role, error) {
			if tk.Value() == "valid_access_token" {
				return userID, role, nil
			}
			return uuid.Nil, domainUser.RoleGhost, errors.New("invalid")
		},
	}
	rtRepo := &mockRefreshTokenRepo{}

	userSvc := userApp.NewService(userRepo, hasher)
	rtRepoSvc := sessionApp.NewService(rtRepo)
	svc := NewService(userSvc, rtRepoSvc, tokenSvc, hasher)

	// Case 1: Success
	uID, r, err := svc.Authorize(tok)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if uID != userID {
		t.Errorf("Expected userID %s, got %s", userID, uID)
	}
	if r != role {
		t.Errorf("Expected role %s, got %s", role, r)
	}

	// Case 2: Error
	invalidTok, _ := tokenDomain.NewToken("invalid")
	_, _, err = svc.Authorize(invalidTok)
	if err == nil {
		t.Fatal("Expected error for invalid access token, got nil")
	}
}
