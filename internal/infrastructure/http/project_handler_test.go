package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	application "github.com/safarislava/typstlab-server/internal/application/project"
	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type mockRepository struct {
	saveFunc     func(ctx context.Context, p *domain.Project) error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Project, error)
}

func (m *mockRepository) Save(ctx context.Context, p *domain.Project) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, p)
	}
	return nil
}

func (m *mockRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
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

func TestProjectHandler_Get_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, err := domain.NewProject(projectID, []uuid.UUID{userID}, "Get Test Project", time.Now())
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	repo := &mockRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			if id == projectID {
				return p, nil
			}
			return nil, nil
		},
	}
	svc := application.NewService(repo)
	handler := NewProjectHandler(svc)

	ctx := context.WithValue(context.Background(), userIDKey, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectID", projectID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String(), nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp JSONProjectResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != projectID.String() {
		t.Errorf("Expected project ID %q, got %q", projectID.String(), resp.ID)
	}
	if resp.Name != "Get Test Project" {
		t.Errorf("Expected project name %q, got %q", "Get Test Project", resp.Name)
	}
}
