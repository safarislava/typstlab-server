package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"

	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
	"github.com/safarislava/typstlab-server/internal/infrastructure/websocket"
)

type mockConnection struct {
	closeCalled bool
}

func (m *mockConnection) Read(ctx context.Context) (coderws.MessageType, []byte, error) {
	<-ctx.Done()
	return 0, nil, fmt.Errorf("ctx done: %w", ctx.Err())
}

func (m *mockConnection) Write(_ context.Context, _ coderws.MessageType, _ []byte) error {
	return nil
}

func (m *mockConnection) Ping(_ context.Context) error {
	return nil
}

func (m *mockConnection) Close(_ coderws.StatusCode, _ string) error {
	m.closeCalled = true
	return nil
}

type mockUserService struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

func (m *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

type mockCollaborationService struct {
	handledClient *websocket.Client
}

func (m *mockCollaborationService) HandleConnection(_ context.Context, client *websocket.Client) {
	m.handledClient = client
}

type mockUpgrader struct {
	upgradeFunc func(w http.ResponseWriter, r *http.Request) (websocket.Connection, error)
}

func (m *mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (websocket.Connection, error) {
	if m.upgradeFunc != nil {
		return m.upgradeFunc(w, r)
	}
	return nil, nil
}

func TestHandler_ServeWS_Unauthorized(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)

	handler.ServeWS(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_ServeWS_ProjectNotFound(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
	req = req.WithContext(middleware.WithUserID(req.Context(), uuid.New()))

	handler.ServeWS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandler_ServeWS_UserNotFound(t *testing.T) {
	t.Parallel()

	mockUser := &mockUserService{
		getByIDFunc: func(_ context.Context, _ uuid.UUID) (*domainUser.User, error) {
			return nil, errors.New("not found in db")
		},
	}

	handler := NewHandler(nil, nil, mockUser, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)

	userID := uuid.New()
	proj, _ := domainProject.NewProject(uuid.New(), []uuid.UUID{userID}, "Project", time.Now())

	ctx := middleware.WithUserID(req.Context(), userID)
	ctx = middleware.WithProject(ctx, proj)
	req = req.WithContext(ctx)

	handler.ServeWS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandler_ServeWS_UpgradeError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	testUser, _ := domainUser.NewUser(userID, "test@test.com", "hash", domainUser.RoleUser)
	mockUser := &mockUserService{
		getByIDFunc: func(_ context.Context, _ uuid.UUID) (*domainUser.User, error) {
			return testUser, nil
		},
	}

	mockUp := &mockUpgrader{
		upgradeFunc: func(_ http.ResponseWriter, _ *http.Request) (websocket.Connection, error) {
			return nil, errors.New("upgrade rejected")
		},
	}

	handler := NewHandler(mockUp, nil, mockUser, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)

	proj, _ := domainProject.NewProject(uuid.New(), []uuid.UUID{userID}, "Project", time.Now())
	ctx := middleware.WithUserID(req.Context(), userID)
	ctx = middleware.WithProject(ctx, proj)
	req = req.WithContext(ctx)

	handler.ServeWS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected recorder to keep status 200 before upgrade write, got %d", rec.Code)
	}
}

func TestHandler_ServeWS_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	testUser, _ := domainUser.NewUser(userID, "test@test.com", "hash", domainUser.RoleUser)
	mockUser := &mockUserService{
		getByIDFunc: func(_ context.Context, _ uuid.UUID) (*domainUser.User, error) {
			return testUser, nil
		},
	}

	conn := &mockConnection{}
	mockUp := &mockUpgrader{
		upgradeFunc: func(_ http.ResponseWriter, _ *http.Request) (websocket.Connection, error) {
			return conn, nil
		},
	}

	hub := websocket.NewHub()
	mockCollab := &mockCollaborationService{}
	handler := NewHandler(mockUp, hub, mockUser, mockCollab)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)

	proj, _ := domainProject.NewProject(uuid.New(), []uuid.UUID{userID}, "Project", time.Now())
	ctx := middleware.WithUserID(req.Context(), userID)
	ctx = middleware.WithProject(ctx, proj)
	req = req.WithContext(ctx)

	handler.ServeWS(rec, req)

	if mockCollab.handledClient == nil {
		t.Fatal("expected collaboration service to handle client")
	}
	if mockCollab.handledClient.UserID() != userID {
		t.Errorf("expected client userID %s, got %s", userID, mockCollab.handledClient.UserID())
	}
	// After ServeWS returns, client is closed and removed from hub
	if hub.RoomsCount() != 0 {
		t.Errorf("expected empty room to be cleaned from hub, got %d", hub.RoomsCount())
	}
}
