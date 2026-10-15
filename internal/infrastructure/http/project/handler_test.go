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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	domain "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

const (
	testInvalidUUID    = "invalid-uuid"
	defaultProjectName = "Project"
	testMemberEmail    = "member@example.com"
)

type mockProjectService struct {
	createFunc       func(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error)
	addMemberFunc    func(ctx context.Context, projectID uuid.UUID, email string) (*projectApp.MemberDTO, error)
	removeMemberFunc func(ctx context.Context, projectID, userID uuid.UUID) error
	listMembersFunc  func(ctx context.Context, projectID uuid.UUID) ([]projectApp.MemberDTO, error)
}

func (m *mockProjectService) Create(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockProjectService) AddMember(ctx context.Context, projectID uuid.UUID, email string) (*projectApp.MemberDTO, error) {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, projectID, email)
	}
	return nil, nil
}

func (m *mockProjectService) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	if m.removeMemberFunc != nil {
		return m.removeMemberFunc(ctx, projectID, userID)
	}
	return nil
}

func (m *mockProjectService) ListMembers(ctx context.Context, projectID uuid.UUID) ([]projectApp.MemberDTO, error) {
	if m.listMembersFunc != nil {
		return m.listMembersFunc(ctx, projectID)
	}
	return nil, nil
}

func TestNewHandler(t *testing.T) {
	t.Parallel()

	projSvc := &mockProjectService{}
	handler := NewHandler(projSvc)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.projectService == nil {
		t.Error("Handler service should not be nil")
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

	if resp.ID != projectID.String() {
		t.Errorf("Expected project ID %q, got %q", projectID.String(), resp.ID)
	}
	if resp.Name != testProjectName {
		t.Errorf("Expected project name %q, got %q", testProjectName, resp.Name)
	}
	if resp.UpdatedAt != expectedTime.Format(time.RFC3339) {
		t.Errorf("Expected updatedAt %q, got %q", expectedTime.Format(time.RFC3339), resp.UpdatedAt)
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&mockProjectService{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_Create_InvalidUUID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&mockProjectService{})
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

	handler := NewHandler(&mockProjectService{})
	reqBody, _ := json.Marshal(jsonCreateRequest{ID: uuid.New().String(), Name: defaultProjectName})
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

	handler := NewHandler(&mockProjectService{})

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

	handler := NewHandler(&mockProjectService{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/projects/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandler_ListMembers(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{userID}, "Members Proj", time.Now())

	svc := &mockProjectService{
		listMembersFunc: func(ctx context.Context, pid uuid.UUID) ([]projectApp.MemberDTO, error) {
			return []projectApp.MemberDTO{
				{ID: userID, Email: testMemberEmail},
			}, nil
		},
	}
	handler := NewHandler(svc)

	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String()+"/members", nil)
	rr := httptest.NewRecorder()

	handler.ListMembers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var members []JSONMemberResponse
	if err := json.NewDecoder(rr.Body).Decode(&members); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}
	if len(members) != 1 || members[0].Email != testMemberEmail {
		t.Errorf("Unexpected members: %v", members)
	}
}

func TestHandler_AddMember(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{userID}, "Add Member Proj", time.Now())

	newUserID := uuid.New()
	svc := &mockProjectService{
		addMemberFunc: func(ctx context.Context, pid uuid.UUID, email string) (*projectApp.MemberDTO, error) {
			if email == "new@example.com" {
				return &projectApp.MemberDTO{ID: newUserID, Email: email}, nil
			}
			return nil, errors.New("user not found")
		},
	}
	handler := NewHandler(svc)

	ctx := middleware.WithProject(context.Background(), p)
	body := bytes.NewBufferString(`{"email":"new@example.com"}`)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/members", body)
	rr := httptest.NewRecorder()

	handler.AddMember(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", rr.Code)
	}
}

func TestHandler_RemoveMember(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{user1, user2}, "Remove Proj", time.Now())

	svc := &mockProjectService{
		removeMemberFunc: func(ctx context.Context, pid, uid uuid.UUID) error {
			if uid == user2 {
				return nil
			}
			return domain.ErrNoUsers
		},
	}
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Delete("/projects/{projectID}/members/{userID}", handler.RemoveMember)

	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/projects/"+projectID.String()+"/members/"+user2.String(), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", rr.Code)
	}
}

func TestHandler_ListMembers_Errors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{uuid.New()}, "Members Proj", time.Now())

	t.Run("MissingContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockProjectService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/projects/123/members", nil)
		rr := httptest.NewRecorder()
		handler.ListMembers(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProjectService{
			listMembersFunc: func(ctx context.Context, pid uuid.UUID) ([]projectApp.MemberDTO, error) {
				return nil, errors.New("list failed")
			},
		}
		handler := NewHandler(svc)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String()+"/members", nil)
		rr := httptest.NewRecorder()
		handler.ListMembers(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})
}

func TestHandler_AddMember_ValidationErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{uuid.New()}, "Add Member Proj", time.Now())

	t.Run("MissingContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockProjectService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/members", nil)
		rr := httptest.NewRecorder()
		handler.AddMember(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockProjectService{})
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/members", bytes.NewBufferString("{bad-json"))
		rr := httptest.NewRecorder()
		handler.AddMember(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockProjectService{})
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/members", bytes.NewBufferString(`{"email":""}`))
		rr := httptest.NewRecorder()
		handler.AddMember(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}

func TestHandler_AddMember_ServiceErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{uuid.New()}, "Add Member Proj", time.Now())

	t.Run("AlreadyMember", func(t *testing.T) {
		t.Parallel()
		svc := &mockProjectService{
			addMemberFunc: func(ctx context.Context, pid uuid.UUID, email string) (*projectApp.MemberDTO, error) {
				return nil, domain.ErrUserAlreadyMember
			},
		}
		handler := NewHandler(svc)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/members", bytes.NewBufferString(`{"email":"dup@example.com"}`))
		rr := httptest.NewRecorder()
		handler.AddMember(rr, req)
		if rr.Code != http.StatusConflict {
			t.Errorf("Expected 409, got %d", rr.Code)
		}
	})

	t.Run("GenericError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProjectService{
			addMemberFunc: func(ctx context.Context, pid uuid.UUID, email string) (*projectApp.MemberDTO, error) {
				return nil, errors.New("cannot add")
			},
		}
		handler := NewHandler(svc)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/members", bytes.NewBufferString(`{"email":"test@example.com"}`))
		rr := httptest.NewRecorder()
		handler.AddMember(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}

func TestHandler_RemoveMember_ValidationErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{uuid.New()}, "Remove Proj", time.Now())

	t.Run("MissingContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockProjectService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/members", nil)
		rr := httptest.NewRecorder()
		handler.RemoveMember(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})

	t.Run("InvalidUserID", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockProjectService{})
		r := chi.NewRouter()
		r.Delete("/members/{userID}", handler.RemoveMember)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/members/invalid-uuid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}

func TestHandler_RemoveMember_DomainErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{uuid.New()}, "Remove Proj", time.Now())
	userID := uuid.New()

	t.Run("LastUser", func(t *testing.T) {
		t.Parallel()
		svc := &mockProjectService{
			removeMemberFunc: func(ctx context.Context, pid, uid uuid.UUID) error {
				return domain.ErrNoUsers
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Delete("/members/{userID}", handler.RemoveMember)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/members/"+userID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})

	t.Run("UserNotInProject", func(t *testing.T) {
		t.Parallel()
		svc := &mockProjectService{
			removeMemberFunc: func(ctx context.Context, pid, uid uuid.UUID) error {
				return domain.ErrUserNotInProject
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Delete("/members/{userID}", handler.RemoveMember)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/members/"+userID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", rr.Code)
		}
	})
}

func TestHandler_RemoveMember_GenericError(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domain.NewProject(projectID, []uuid.UUID{uuid.New()}, "Remove Proj", time.Now())
	userID := uuid.New()

	svc := &mockProjectService{
		removeMemberFunc: func(ctx context.Context, pid, uid uuid.UUID) error {
			return errors.New("cannot remove")
		},
	}
	handler := NewHandler(svc)
	r := chi.NewRouter()
	r.Delete("/members/{userID}", handler.RemoveMember)
	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/members/"+userID.String(), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rr.Code)
	}
}
