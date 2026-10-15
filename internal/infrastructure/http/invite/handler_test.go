package invite

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

	domainInvite "github.com/safarislava/typstlab-server/internal/domain/invite"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

type mockService struct {
	createInviteFunc func(ctx context.Context, pid, uid uuid.UUID, exp time.Time) (*domainInvite.Invite, error)
	listInvitesFunc  func(ctx context.Context, pid uuid.UUID) ([]*domainInvite.Invite, error)
	revokeInviteFunc func(ctx context.Context, pid, iid uuid.UUID) error
	acceptInviteFunc func(ctx context.Context, code string, uid uuid.UUID) (*domainProject.Project, error)
}

func (m *mockService) CreateInvite(ctx context.Context, pid, uid uuid.UUID, exp time.Time) (*domainInvite.Invite, error) {
	if m.createInviteFunc != nil {
		return m.createInviteFunc(ctx, pid, uid, exp)
	}
	return nil, nil
}

func (m *mockService) ListInvites(ctx context.Context, pid uuid.UUID) ([]*domainInvite.Invite, error) {
	if m.listInvitesFunc != nil {
		return m.listInvitesFunc(ctx, pid)
	}
	return nil, nil
}

func (m *mockService) RevokeInvite(ctx context.Context, pid, iid uuid.UUID) error {
	if m.revokeInviteFunc != nil {
		return m.revokeInviteFunc(ctx, pid, iid)
	}
	return nil
}

func (m *mockService) AcceptInvite(ctx context.Context, code string, uid uuid.UUID) (*domainProject.Project, error) {
	if m.acceptInviteFunc != nil {
		return m.acceptInviteFunc(ctx, code, uid)
	}
	return nil, nil
}

const testInviteSecret = "secret123"

func TestHandler_CreateInvite(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Invites Proj", time.Now())
	invID := uuid.New()
	inv, _ := domainInvite.NewInvite(invID, projectID, userID, testInviteSecret, time.Now().Add(24*time.Hour), time.Now())

	svc := &mockService{
		createInviteFunc: func(ctx context.Context, pid, uid uuid.UUID, exp time.Time) (*domainInvite.Invite, error) {
			return inv, nil
		},
	}
	handler := NewHandler(svc)

	ctx := middleware.WithProject(context.Background(), p)
	ctx = middleware.WithUserID(ctx, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/invites", nil)
	rr := httptest.NewRecorder()
	handler.CreateInvite(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("CreateInvite expected status 201, got %d", rr.Code)
	}

	var resp JSONInviteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Code != testInviteSecret {
		t.Errorf("Unexpected invite code: %s", resp.Code)
	}
}

func TestHandler_CreateInvite_ContextErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Proj", time.Now())

	t.Run("MissingProjectContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/invites", nil)
		rr := httptest.NewRecorder()
		handler.CreateInvite(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})

	t.Run("MissingUserContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites", nil)
		rr := httptest.NewRecorder()
		handler.CreateInvite(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})
}

func TestHandler_CreateInvite_ServiceAndBody(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Proj", time.Now())

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			createInviteFunc: func(ctx context.Context, pid, uid uuid.UUID, exp time.Time) (*domainInvite.Invite, error) {
				return nil, errors.New("boom")
			},
		}
		handler := NewHandler(svc)
		ctx := middleware.WithProject(context.Background(), p)
		ctx = middleware.WithUserID(ctx, userID)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites", nil)
		rr := httptest.NewRecorder()
		handler.CreateInvite(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})

	t.Run("WithExpiresAtBody", func(t *testing.T) {
		t.Parallel()
		inv, _ := domainInvite.NewInvite(uuid.New(), projectID, userID, "exp123", time.Now().Add(48*time.Hour), time.Now())
		svc := &mockService{
			createInviteFunc: func(ctx context.Context, pid, uid uuid.UUID, exp time.Time) (*domainInvite.Invite, error) {
				return inv, nil
			},
		}
		handler := NewHandler(svc)
		ctx := middleware.WithProject(context.Background(), p)
		ctx = middleware.WithUserID(ctx, userID)
		body := bytes.NewBufferString(`{"expires_at":"2030-01-01T00:00:00Z"}`)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites", body)
		rr := httptest.NewRecorder()
		handler.CreateInvite(rr, req)
		if rr.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", rr.Code)
		}
	})
}

func TestHandler_ListInvites(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Invites Proj", time.Now())
	invID := uuid.New()
	inv, _ := domainInvite.NewInvite(invID, projectID, userID, testInviteSecret, time.Now().Add(24*time.Hour), time.Now())

	svc := &mockService{
		listInvitesFunc: func(ctx context.Context, pid uuid.UUID) ([]*domainInvite.Invite, error) {
			return []*domainInvite.Invite{inv}, nil
		},
	}
	handler := NewHandler(svc)

	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String()+"/invites", nil)
	rr := httptest.NewRecorder()
	handler.ListInvites(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ListInvites expected status 200, got %d", rr.Code)
	}

	var resp []JSONInviteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Code != testInviteSecret {
		t.Errorf("Unexpected invites: %v", resp)
	}
}

func TestHandler_ListInvites_Errors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Proj", time.Now())

	t.Run("MissingProjectContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/invites", nil)
		rr := httptest.NewRecorder()
		handler.ListInvites(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			listInvitesFunc: func(ctx context.Context, pid uuid.UUID) ([]*domainInvite.Invite, error) {
				return nil, errors.New("list failed")
			},
		}
		handler := NewHandler(svc)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/invites", nil)
		rr := httptest.NewRecorder()
		handler.ListInvites(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})
}

func TestHandler_RevokeInvite(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Invites Proj", time.Now())
	invID := uuid.New()

	svc := &mockService{
		revokeInviteFunc: func(ctx context.Context, pid, iid uuid.UUID) error {
			return nil
		},
	}
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Delete("/projects/{projectID}/invites/{inviteID}", handler.RevokeInvite)

	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/projects/"+projectID.String()+"/invites/"+invID.String(), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("RevokeInvite expected status 204, got %d", rr.Code)
	}
}

func TestHandler_RevokeInvite_ContextErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Proj", time.Now())

	t.Run("MissingProjectContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/invites", nil)
		rr := httptest.NewRecorder()
		handler.RevokeInvite(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", rr.Code)
		}
	})

	t.Run("InvalidInviteID", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		r := chi.NewRouter()
		r.Delete("/invites/{inviteID}", handler.RevokeInvite)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/invites/invalid-uuid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}

func TestHandler_RevokeInvite_ServiceErrors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Proj", time.Now())
	invID := uuid.New()

	t.Run("InviteNotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			revokeInviteFunc: func(ctx context.Context, pid, iid uuid.UUID) error {
				return domainInvite.ErrInviteNotFound
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Delete("/invites/{inviteID}", handler.RevokeInvite)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/invites/"+invID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", rr.Code)
		}
	})

	t.Run("GenericError", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			revokeInviteFunc: func(ctx context.Context, pid, iid uuid.UUID) error {
				return errors.New("cannot delete")
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Delete("/invites/{inviteID}", handler.RevokeInvite)
		ctx := middleware.WithProject(context.Background(), p)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/invites/"+invID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}

func TestHandler_AcceptInvite(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Invites Proj", time.Now())

	svc := &mockService{
		acceptInviteFunc: func(ctx context.Context, code string, uid uuid.UUID) (*domainProject.Project, error) {
			return p, nil
		},
	}
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Post("/invites/{code}/accept", handler.AcceptInvite)

	ctx := middleware.WithUserID(context.Background(), userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites/"+testInviteSecret+"/accept", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("AcceptInvite expected status 200, got %d", rr.Code)
	}
}

func TestHandler_AcceptInvite_ParamErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("MissingUserContext", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/invites/test/accept", nil)
		rr := httptest.NewRecorder()
		handler.AcceptInvite(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("MissingCodeParam", func(t *testing.T) {
		t.Parallel()
		handler := NewHandler(&mockService{})
		ctx := middleware.WithUserID(context.Background(), userID)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites//accept", nil)
		rr := httptest.NewRecorder()
		handler.AcceptInvite(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}

func TestHandler_AcceptInvite_ServiceErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("InviteNotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			acceptInviteFunc: func(ctx context.Context, code string, uid uuid.UUID) (*domainProject.Project, error) {
				return nil, domainInvite.ErrInviteNotFound
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Post("/invites/{code}/accept", handler.AcceptInvite)
		ctx := middleware.WithUserID(context.Background(), userID)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites/badcode/accept", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", rr.Code)
		}
	})

	t.Run("InviteExpired", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			acceptInviteFunc: func(ctx context.Context, code string, uid uuid.UUID) (*domainProject.Project, error) {
				return nil, domainInvite.ErrInviteExpired
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Post("/invites/{code}/accept", handler.AcceptInvite)
		ctx := middleware.WithUserID(context.Background(), userID)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites/oldcode/accept", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusGone {
			t.Errorf("Expected 410, got %d", rr.Code)
		}
	})

	t.Run("GenericError", func(t *testing.T) {
		t.Parallel()
		svc := &mockService{
			acceptInviteFunc: func(ctx context.Context, code string, uid uuid.UUID) (*domainProject.Project, error) {
				return nil, errors.New("accept error")
			},
		}
		handler := NewHandler(svc)
		r := chi.NewRouter()
		r.Post("/invites/{code}/accept", handler.AcceptInvite)
		ctx := middleware.WithUserID(context.Background(), userID)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/invites/code/accept", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})
}
