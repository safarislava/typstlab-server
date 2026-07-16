package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type mockUserRepo struct {
	saveFunc        func(ctx context.Context, u *domainUser.User) error
	findByEmailFunc func(ctx context.Context, email string) (*domainUser.User, error)
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

func (m *mockUserRepo) FindByID(context.Context, uuid.UUID) (*domainUser.User, error) {
	return nil, nil
}

type mockUserHasher struct {
	hashFunc    func(password string) (string, error)
	compareFunc func(hashedPassword, password string) error
}

func (m *mockUserHasher) Hash(password string) (string, error) {
	if m.hashFunc != nil {
		return m.hashFunc(password)
	}
	return "hashed_password", nil
}

func (m *mockUserHasher) Compare(hashedPassword, password string) error {
	if m.compareFunc != nil {
		return m.compareFunc(hashedPassword, password)
	}
	return nil
}

type mockUserTokenService struct {
	generateFunc func(userID uuid.UUID, role domainUser.Role) (token.Token, error)
	validateFunc func(t token.Token) (uuid.UUID, domainUser.Role, error)
}

func (m *mockUserTokenService) Generate(userID uuid.UUID, role domainUser.Role) (token.Token, error) {
	if m.generateFunc != nil {
		return m.generateFunc(userID, role)
	}
	t, err := token.NewToken("mock-token")
	if err != nil {
		return token.Token{}, fmt.Errorf("failed to create domain token: %w", err)
	}
	return t, nil
}

func (m *mockUserTokenService) Validate(t token.Token) (uuid.UUID, domainUser.Role, error) {
	if m.validateFunc != nil {
		return m.validateFunc(t)
	}
	return uuid.Nil, domainUser.RoleGhost, errors.New("invalid")
}

func TestUserHandler_Register(t *testing.T) {
	t.Parallel()

	repo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			return nil, errors.New("not found")
		},
	}
	hasher := &mockUserHasher{}
	tokenSvc := &mockUserTokenService{}
	svc := user.NewService(repo, hasher, tokenSvc)
	handler := NewUserHandler(svc)

	// Case 1: Invalid JSON
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()
	handler.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	// Case 2: Success
	body := `{"email":"new@example.com","password":"password","role":"user"}`
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	handler.Register(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d, body %s", rr.Code, rr.Body.String())
	}

	// Case 3: Registration error
	body = `{"email":"invalid","password":"password"}`
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	handler.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestUserHandler_Login(t *testing.T) {
	t.Parallel()

	u, _ := domainUser.NewUser(uuid.New(), "login@example.com", "hashed_password", domainUser.RoleUser)
	repo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			if email == "login@example.com" {
				return u, nil
			}
			return nil, errors.New("not found")
		},
	}
	hasher := &mockUserHasher{}
	tokenSvc := &mockUserTokenService{}
	svc := user.NewService(repo, hasher, tokenSvc)
	handler := NewUserHandler(svc)

	// Case 1: Invalid JSON
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()
	handler.Login(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	// Case 2: Success
	body := `{"email":"login@example.com","password":"password"}`
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	handler.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d, body %s", rr.Code, rr.Body.String())
	}

	// Case 3: Login failure
	body = `{"email":"wrong@example.com","password":"password"}`
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	handler.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}
