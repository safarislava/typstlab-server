package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	domain "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

const (
	testInvalidUUID    = "invalid-uuid"
	defaultProjectName = "Project"
)

type mockProjectService struct {
	createFunc func(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error)
}

func (m *mockProjectService) Create(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func TestNewHandler(t *testing.T) {
	t.Parallel()

	svc := &mockProjectService{}
	handler := NewHandler(svc)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.service != svc {
		t.Error("Handler service mismatch")
	}
}

func TestHandler_Create_Success(t *testing.T) {
	t.Parallel()

	const testProjectName = "HTTP Test Project"
	projectID := uuid.New()
	expectedTime := time.Now()

	svc := &mockProjectService{
		createFunc: func(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error) {
			return &projectApp.CreateResponse{
				ID:        req.ID,
				Name:      req.Name,
				UpdatedAt: expectedTime,
			}, nil
		},
	}
	handler := NewHandler(svc)

	reqBody, _ := json.Marshal(jsonCreateRequest{ID: projectID.String(), Name: testProjectName})
	ctx := middleware.WithUserID(context.Background(), uuid.New())
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
	if resp.ID != projectID.String() {
		t.Errorf("Expected ID %q, got %q", projectID.String(), resp.ID)
	}
	if resp.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be populated")
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be application/json, got %q", rr.Header().Get("Content-Type"))
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	t.Parallel()

	svc := &mockProjectService{}
	handler := NewHandler(svc)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_Create_InvalidID(t *testing.T) {
	t.Parallel()

	svc := &mockProjectService{}
	handler := NewHandler(svc)

	reqBody, _ := json.Marshal(jsonCreateRequest{ID: testInvalidUUID, Name: defaultProjectName})
	ctx := middleware.WithUserID(context.Background(), uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_Create_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := &mockProjectService{}
	handler := NewHandler(svc)

	projectID := uuid.New().String()
	reqBody, _ := json.Marshal(jsonCreateRequest{ID: projectID, Name: defaultProjectName})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &mockProjectService{
		createFunc: func(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error) {
			return nil, errors.New("service failure")
		},
	}
	handler := NewHandler(svc)

	reqBody, _ := json.Marshal(jsonCreateRequest{ID: uuid.New().String(), Name: defaultProjectName})
	ctx := middleware.WithUserID(context.Background(), uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if rr.Body.String() != "service failure" {
		t.Errorf("Expected body 'service failure', got %q", rr.Body.String())
	}
}

func TestHandler_Get_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, err := domain.NewProject(projectID, []uuid.UUID{userID}, "Get Test Project", time.Now())
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	svc := &mockProjectService{}
	handler := NewHandler(svc)

	ctx := middleware.WithUserID(context.Background(), userID)
	ctx = middleware.WithProject(ctx, p)

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

func TestHandler_Get_MissingContext(t *testing.T) {
	t.Parallel()

	svc := &mockProjectService{}
	handler := NewHandler(svc)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/projects/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
