package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

const (
	testPassword   = "password"
	testQueryEmail = "test@example.com"
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

func TestService_Register_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("not found")
		},
	}
	hasher := &mockHasher{}

	svc := NewService(repo, hasher)

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

	svc := NewService(repo, hasher)

	req := RegisterRequest{
		Email:    "exist@example.com",
		Password: testPassword,
	}

	_, err := svc.Register(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error user already exists, got nil")
	}
}

func TestService_GetByEmail_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	u, _ := domain.NewUser(userID, testQueryEmail, "hash", domain.RoleUser)
	repo := &mockRepository{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			if email == testQueryEmail {
				return u, nil
			}
			return nil, errors.New("not found")
		},
	}
	hasher := &mockHasher{}
	svc := NewService(repo, hasher)

	found, err := svc.GetByEmail(context.Background(), testQueryEmail)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if found.ID() != userID {
		t.Errorf("Expected ID %s, got %s", userID, found.ID())
	}
}

func TestService_GetByID_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	u, _ := domain.NewUser(userID, testQueryEmail, "hash", domain.RoleUser)
	repo := &mockRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			if id == userID {
				return u, nil
			}
			return nil, errors.New("not found")
		},
	}
	hasher := &mockHasher{}
	svc := NewService(repo, hasher)

	found, err := svc.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if found.Email() != testQueryEmail {
		t.Errorf("Expected email %s, got %s", testQueryEmail, found.Email())
	}
}
