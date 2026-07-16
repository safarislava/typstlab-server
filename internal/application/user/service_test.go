package user

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

const (
	testPassword   = "password"
	testLoginEmail = "login@example.com"
)

type mockRepository struct {
	saveFunc        func(ctx context.Context, u *domain.User) error
	findByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	findByIDFunc    func(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

func (m *mockRepository) Save(ctx context.Context, u *domain.User) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, u)
	}
	return nil
}

func (m *mockRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(ctx, email)
	}
	return nil, errors.New("not found")
}

func (m *mockRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
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
	generateFunc func(userID uuid.UUID, role domain.Role) (token.Token, error)
	validateFunc func(t token.Token) (uuid.UUID, domain.Role, error)
}

func (m *mockTokenService) Generate(userID uuid.UUID, role domain.Role) (token.Token, error) {
	if m.generateFunc != nil {
		return m.generateFunc(userID, role)
	}
	t, err := token.NewToken("token_" + userID.String())
	if err != nil {
		return token.Token{}, fmt.Errorf("failed to create mock token: %w", err)
	}
	return t, nil
}

func (m *mockTokenService) Validate(t token.Token) (uuid.UUID, domain.Role, error) {
	if m.validateFunc != nil {
		return m.validateFunc(t)
	}
	return uuid.Nil, domain.RoleGhost, errors.New("invalid")
}

func TestService_Register_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("not found")
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}

	svc := NewService(repo, hasher, tokenSvc)

	const testEmail = "new@example.com"
	req := RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
	}

	resp, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Email != testEmail {
		t.Errorf("Expected email %q, got %q", testEmail, resp.Email)
	}
	if resp.Role != "user" {
		t.Errorf("Expected default role 'user', got %q", resp.Role)
	}
}

func TestService_Register_AlreadyExists(t *testing.T) {
	t.Parallel()

	u, _ := domain.NewUser(uuid.New(), "exist@example.com", "hash", domain.RoleUser)
	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}

	svc := NewService(repo, hasher, tokenSvc)

	req := RegisterRequest{
		Email:    "exist@example.com",
		Password: testPassword,
	}

	_, err := svc.Register(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error user already exists, got nil")
	}
}

func TestService_Login_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	u, _ := domain.NewUser(userID, testLoginEmail, "hashed_secret", domain.RoleUser)
	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}

	svc := NewService(repo, hasher, tokenSvc)

	req := LoginRequest{
		Email:    testLoginEmail,
		Password: "secret",
	}

	resp, err := svc.Login(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedToken := "token_" + userID.String()
	if resp.Token.Value() != expectedToken {
		t.Errorf("Expected token %q, got %q", expectedToken, resp.Token.Value())
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("not found")
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}

	svc := NewService(repo, hasher, tokenSvc)

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

	u, _ := domain.NewUser(uuid.New(), testLoginEmail, "hashed_secret", domain.RoleUser)
	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{
		compareFunc: func(hashedPassword, password string) error {
			return errors.New("mismatch")
		},
	}
	tokenSvc := &mockTokenService{}

	svc := NewService(repo, hasher, tokenSvc)

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

	u, _ := domain.NewUser(uuid.New(), testLoginEmail, "hashed_secret", domain.RoleUser)
	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{
		generateFunc: func(userID uuid.UUID, role domain.Role) (token.Token, error) {
			return token.Token{}, errors.New("failed to generate")
		},
	}

	svc := NewService(repo, hasher, tokenSvc)

	req := LoginRequest{
		Email:    testLoginEmail,
		Password: testPassword,
	}

	_, err := svc.Login(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error token generation failure, got nil")
	}
}
