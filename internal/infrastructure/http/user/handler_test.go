package user

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	appUser "github.com/safarislava/typstlab-server/internal/application/user"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type mockUserService struct {
	registerFunc func(ctx context.Context, req appUser.RegisterRequest) (*appUser.RegisterResponse, error)
}

func (m *mockUserService) Register(ctx context.Context, req appUser.RegisterRequest) (*appUser.RegisterResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, req)
	}
	return nil, errors.New("registration failed")
}

func TestUserHandler_Register(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockUserService{
		registerFunc: func(ctx context.Context, req appUser.RegisterRequest) (*appUser.RegisterResponse, error) {
			if req.Email == "new@example.com" {
				return &appUser.RegisterResponse{
					ID:    userID,
					Email: req.Email,
					Role:  domainUser.RoleUser,
				}, nil
			}
			return nil, errors.New("invalid email")
		},
	}
	handler := NewHandler(svc)

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
