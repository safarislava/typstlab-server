package invite

import (
	"context"
	"encoding/json"
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
