package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safarislava/typstlab-server/internal/application/user"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

func TestUserHandler_Register(t *testing.T) {
	t.Parallel()

	repo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			return nil, errors.New("not found")
		},
	}
	hasher := &mockUserHasher{}
	svc := user.NewService(repo, hasher)
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
