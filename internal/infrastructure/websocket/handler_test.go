package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"

	collabApp "github.com/safarislava/typstlab-server/internal/application/collaboration"
	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

const (
	testKeyFileID = "file_id"
	testKeyDelta  = "delta"
)

type mockConnection struct {
	readChan  chan []byte
	writeChan chan []byte
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		readChan:  make(chan []byte, 10),
		writeChan: make(chan []byte, 10),
	}
}

func (m *mockConnection) Read(ctx context.Context) (coderws.MessageType, []byte, error) {
	select {
	case <-ctx.Done():
		return 0, nil, fmt.Errorf("ctx done: %w", ctx.Err())
	case data, ok := <-m.readChan:
		if !ok {
			return 0, nil, io.EOF
		}
		return coderws.MessageText, data, nil
	}
}

func (m *mockConnection) Write(_ context.Context, _ coderws.MessageType, p []byte) error {
	m.writeChan <- p
	return nil
}

func (m *mockConnection) Ping(_ context.Context) error {
	return nil
}

func (m *mockConnection) Close(_ coderws.StatusCode, _ string) error {
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
	syncInitFunc            func(ctx context.Context, projectID uuid.UUID, req collabApp.SyncInitRequest) (*syncApp.Response, error)
	applyCRDTUpdateFunc     func(ctx context.Context, fileID uuid.UUID, delta []byte) error
	applyMetadataUpdateFunc func(ctx context.Context, projectID uuid.UUID, delta []byte) error
}

func (m *mockCollaborationService) SyncInit(
	ctx context.Context,
	projectID uuid.UUID,
	req collabApp.SyncInitRequest,
) (*syncApp.Response, error) {
	if m.syncInitFunc != nil {
		return m.syncInitFunc(ctx, projectID, req)
	}
	return nil, nil
}

func (m *mockCollaborationService) ApplyCRDTUpdate(ctx context.Context, fileID uuid.UUID, delta []byte) error {
	if m.applyCRDTUpdateFunc != nil {
		return m.applyCRDTUpdateFunc(ctx, fileID, delta)
	}
	return nil
}

func (m *mockCollaborationService) ApplyMetadataUpdate(ctx context.Context, projectID uuid.UUID, delta []byte) error {
	if m.applyMetadataUpdateFunc != nil {
		return m.applyMetadataUpdateFunc(ctx, projectID, delta)
	}
	return nil
}

type mockUpgrader struct {
	upgradeFunc func(w http.ResponseWriter, r *http.Request) (Connection, error)
}

func (m *mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Connection, error) {
	if m.upgradeFunc != nil {
		return m.upgradeFunc(w, r)
	}
	return nil, nil
}

func pushTestMessage(t *testing.T, conn *mockConnection, msg Message) {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}
	conn.readChan <- data
}

func readClientMessage(t *testing.T, client *Client) Message {
	t.Helper()
	select {
	case data := <-client.OutgoingMessages():
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		return msg
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for client message")
		return Message{}
	}
}

type testEnv struct {
	handler *Handler
	collab  *mockCollaborationService
	hub     *Hub
	userID  uuid.UUID
	proj    *domainProject.Project
	conn    *mockConnection
}

func setupTestHandler(t *testing.T) testEnv {
	t.Helper()
	userID := uuid.New()
	testUser, _ := domainUser.NewUser(userID, "test@test.com", "hash", domainUser.RoleUser)
	mockUser := &mockUserService{
		getByIDFunc: func(_ context.Context, _ uuid.UUID) (*domainUser.User, error) {
			return testUser, nil
		},
	}

	conn := newMockConnection()
	mockUp := &mockUpgrader{
		upgradeFunc: func(_ http.ResponseWriter, _ *http.Request) (Connection, error) {
			return conn, nil
		},
	}

	hub := NewHub()
	mockCollab := &mockCollaborationService{}
	handler := NewHandler(mockUp, hub, mockUser, mockCollab)
	projectID := uuid.New()
	proj, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Project", time.Now())

	return testEnv{
		handler: handler,
		collab:  mockCollab,
		hub:     hub,
		userID:  userID,
		proj:    proj,
		conn:    conn,
	}
}

func waitForActiveRoom(t *testing.T, hub *Hub, projectID uuid.UUID) *Room {
	t.Helper()
	for i := 0; i < 20; i++ {
		r, ok := hub.GetRoom(projectID)
		if ok && r.ClientsCount() > 0 {
			return r
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected room to have active client")
	return nil
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
		upgradeFunc: func(_ http.ResponseWriter, _ *http.Request) (Connection, error) {
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
		t.Errorf("expected recorder status 200, got %d", rec.Code)
	}
}

func TestHandler_ServeWS_CollaborationFlow(t *testing.T) {
	t.Parallel()

	env := setupTestHandler(t)
	fileID := uuid.New()

	crdtApplied := false
	env.collab.applyCRDTUpdateFunc = func(_ context.Context, id uuid.UUID, delta []byte) error {
		if id == fileID && string(delta) == "crdt-update" {
			crdtApplied = true
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ws", http.NoBody)
	req = req.WithContext(middleware.WithUserID(middleware.WithProject(ctx, env.proj), env.userID))

	go env.handler.ServeWS(rec, req)

	room := waitForActiveRoom(t, env.hub, env.proj.ID())
	peerClient := NewClient(uuid.New(), "peer@test.com", env.proj.ID(), newMockConnection(), room)
	room.Join(peerClient)
	<-peerClient.OutgoingMessages() // drain snapshot

	// 1. Ping
	pingMsg, _ := NewMessage(domainCollab.MsgPing, nil)
	pushTestMessage(t, env.conn, pingMsg)

	// 2. CRDT Update
	crdtMsg, _ := NewMessage(domainCollab.MsgCRDTUpdate, map[string]any{
		testKeyFileID: fileID.String(),
		testKeyDelta:  "Y3JkdC11cGRhdGU=",
	})
	pushTestMessage(t, env.conn, crdtMsg)

	peerMsg := readClientMessage(t, peerClient)
	verifyCRDTBroadcast(t, peerMsg, env.userID)
	if !crdtApplied {
		t.Error("expected applyCRDTUpdate to be invoked")
	}

	// 3. Awareness Update
	awarenessMsg, _ := NewMessage(domainCollab.MsgAwarenessUpdate, map[string]any{
		testKeyFileID: fileID.String(),
		"cursor": map[string]int{
			"anchor": 5,
			"head":   10,
		},
	})
	pushTestMessage(t, env.conn, awarenessMsg)

	peerAwareness := readClientMessage(t, peerClient)
	if peerAwareness.Type != domainCollab.MsgAwarenessUpdate {
		t.Errorf("expected peer to receive awareness_update, got %s", peerAwareness.Type)
	}

	waitForSelfAwarenessUpdate(t, env.conn)
	cancel()
}

func waitForSelfAwarenessUpdate(t *testing.T, conn *mockConnection) {
	t.Helper()
	for {
		select {
		case data := <-conn.writeChan:
			var selfMsg Message
			if err := json.Unmarshal(data, &selfMsg); err == nil && selfMsg.Type == domainCollab.MsgAwarenessUpdate {
				return
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timed out waiting for self awareness_update")
			return
		}
	}
}

func verifyCRDTBroadcast(t *testing.T, peerMsg Message, expectedSender uuid.UUID) {
	t.Helper()
	if peerMsg.Type != domainCollab.MsgCRDTUpdate {
		t.Errorf("expected peer to receive crdt_update, got %s", peerMsg.Type)
	}
	var broadcast crdtBroadcastPayload
	if err := json.Unmarshal(peerMsg.Payload, &broadcast); err != nil {
		t.Fatalf("failed to unmarshal broadcast payload: %v", err)
	}
	if broadcast.SenderID != expectedSender {
		t.Errorf("expected sender_id %s, got %s", expectedSender, broadcast.SenderID)
	}
}
