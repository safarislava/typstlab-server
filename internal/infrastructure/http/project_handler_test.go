package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	application "github.com/safarislava/typstlab-server/internal/application/project"
	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type mockRepository struct {
	saveFunc func(ctx context.Context, p *domain.Project) error
}

func (m *mockRepository) Save(ctx context.Context, p *domain.Project) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, p)
	}
	return nil
}

func (m *mockRepository) FindByID(context.Context, uuid.UUID) (*domain.Project, error) {
	return nil, nil
}

func TestNewProjectHandler(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := application.NewService(repo)
	handler := NewProjectHandler(svc)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.service != svc {
		t.Error("Handler service mismatch")
	}
}

func TestProjectHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := application.NewService(repo)
	handler := NewProjectHandler(svc)

	const testProjectName = "HTTP Test Project"
	reqBody, _ := json.Marshal(jsonCreateRequest{Name: testProjectName})
	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}

	var resp JSONCreateResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Name != testProjectName {
		t.Errorf("Expected response name %q, got %q", testProjectName, resp.Name)
	}
	if resp.ID == "" {
		t.Error("Expected ID to be populated")
	}
	if resp.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be populated")
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be application/json, got %q", rr.Header().Get("Content-Type"))
	}
}

func TestProjectHandler_Create_InvalidJSON(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := application.NewService(repo)
	handler := NewProjectHandler(svc)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestProjectHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := application.NewService(repo)
	handler := NewProjectHandler(svc)

	reqBody, _ := json.Marshal(jsonCreateRequest{Name: ""})
	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "failed to create domain project") {
		t.Errorf("Expected validation error in body, got: %s", rr.Body.String())
	}
}
