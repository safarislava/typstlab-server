package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

const (
	testPassword = "password"
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
	generateFunc func(userID uuid.UUID, role domain.Role) (string, error)
	validateFunc func(token string) (uuid.UUID, domain.Role, error)
}

func (m *mockTokenService) Generate(userID uuid.UUID, role domain.Role) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(userID, role)
	}
	return "token_" + userID.String(), nil
}

func (m *mockTokenService) Validate(token string) (uuid.UUID, domain.Role, error) {
	if m.validateFunc != nil {
		return m.validateFunc(token)
	}
	return uuid.Nil, "", errors.New("invalid")
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
	u, _ := domain.NewUser(userID, "login@example.com", "hashed_secret", domain.RoleUser)
	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return u, nil
		},
	}
	hasher := &mockHasher{}
	tokenSvc := &mockTokenService{}

	svc := NewService(repo, hasher, tokenSvc)

	req := LoginRequest{
		Email:    "login@example.com",
		Password: "secret",
	}

	resp, err := svc.Login(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedToken := "token_" + userID.String()
	if resp.Token != expectedToken {
		t.Errorf("Expected token %q, got %q", expectedToken, resp.Token)
	}
}
