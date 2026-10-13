package collaboration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	"github.com/safarislava/typstlab-server/internal/infrastructure/websocket"
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
		return 0, nil, fmt.Errorf("context done: %w", ctx.Err())
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

type mockSyncService struct {
	syncFunc             func(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error)
	applyFileChangesFunc func(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
}

func (m *mockSyncService) Sync(
	ctx context.Context,
	projectID uuid.UUID,
	req *syncApp.Request,
) (*syncApp.Response, error) {
	if m.syncFunc != nil {
		return m.syncFunc(ctx, projectID, req)
	}
	return nil, nil
}

func (m *mockSyncService) ApplyFileChanges(
	ctx context.Context,
	req syncApp.ApplyFileChangesRequest,
) (*domainFile.TypstFile, error) {
	if m.applyFileChangesFunc != nil {
		return m.applyFileChangesFunc(ctx, req)
	}
	return nil, nil
}

func setupTestSetup(t *testing.T) (*Service, *mockSyncService, *websocket.Room, *websocket.Client, *mockConnection) {
	t.Helper()
	mockSync := &mockSyncService{}
	service := NewService(mockSync)

	projectID := uuid.New()
	room := websocket.NewRoom(projectID)

	mockConn := newMockConnection()
	userID := uuid.New()
	client := websocket.NewClient(userID, "tester@test.com", projectID, mockConn, room)
	room.Join(client)

	// Drain initial awareness snapshot
	select {
	case <-client.OutgoingMessages():
	case <-time.After(50 * time.Millisecond):
		t.Fatal("timed out waiting for awareness snapshot")
	}

	return service, mockSync, room, client, mockConn
}

func pushMessage(t *testing.T, conn *mockConnection, msg websocket.Message) {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}
	conn.readChan <- data
}

func readOutgoingMessage(t *testing.T, client *websocket.Client) websocket.Message {
	t.Helper()
	select {
	case data := <-client.OutgoingMessages():
		var msg websocket.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		return msg
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for outgoing message")
		return websocket.Message{}
	}
}

func TestService_HandlePing(t *testing.T) {
	t.Parallel()
	service, _, _, client, mockConn := setupTestSetup(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go service.HandleConnection(ctx, client)

	pingMsg, _ := websocket.NewMessage(websocket.MsgPing, nil)
	pushMessage(t, mockConn, pingMsg)

	resp := readOutgoingMessage(t, client)
	if resp.Type != websocket.MsgPong {
		t.Errorf("expected %s, got %s", websocket.MsgPong, resp.Type)
	}

	cancel()
}

func TestService_HandleSyncInit(t *testing.T) {
	t.Parallel()
	service, mockSync, _, client, mockConn := setupTestSetup(t)

	fileID := uuid.New()
	mockSync.syncFunc = func(
		_ context.Context,
		_ uuid.UUID,
		req *syncApp.Request,
	) (*syncApp.Response, error) {
		if string(req.MetadataDelta) != "meta-delta" {
			t.Errorf("unexpected metadata delta: %s", string(req.MetadataDelta))
		}
		return &syncApp.Response{
			MetadataDelta: []byte("server-meta"),
			Instructions: []syncApp.Instruction{
				{
					Action: syncApp.ActionApplyChanges,
					FileID: fileID,
					Delta:  []byte("file-delta"),
				},
			},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.HandleConnection(ctx, client)

	initPayload := map[string]any{
		"metadata_delta":        "bWV0YS1kZWx0YQ==",
		"metadata_state_vector": "c3RhdGUtdmVjdG9y",
		"content_vectors": map[string]string{
			fileID.String(): "ZmlsZS12ZWN0b3I=",
		},
	}
	initMsg, _ := websocket.NewMessage(websocket.MsgSyncInit, initPayload)
	pushMessage(t, mockConn, initMsg)

	resp := readOutgoingMessage(t, client)
	if resp.Type != websocket.MsgSyncResponse {
		t.Fatalf("expected %s, got %s", websocket.MsgSyncResponse, resp.Type)
	}

	var respPayload syncResponsePayload
	if err := json.Unmarshal(resp.Payload, &respPayload); err != nil {
		t.Fatalf("failed to unmarshal sync response payload: %v", err)
	}
	if string(respPayload.MetadataDelta) != "server-meta" {
		t.Errorf("expected metadata_delta server-meta, got %s", string(respPayload.MetadataDelta))
	}
	if len(respPayload.Instructions) != 1 || respPayload.Instructions[0].FileID != fileID {
		t.Errorf("unexpected instructions: %+v", respPayload.Instructions)
	}

	cancel()
}

func TestService_HandleCRDTUpdate(t *testing.T) {
	t.Parallel()
	service, mockSync, room, client1, conn1 := setupTestSetup(t)

	client2 := websocket.NewClient(uuid.New(), "peer@test.com", room.ProjectID(), newMockConnection(), room)
	room.Join(client2)
	<-client2.OutgoingMessages() // drain snapshot

	applied := false
	fileID := uuid.New()
	mockSync.applyFileChangesFunc = func(
		_ context.Context,
		req syncApp.ApplyFileChangesRequest,
	) (*domainFile.TypstFile, error) {
		if req.FileID == fileID && string(req.Delta) == "crdt-diff" {
			applied = true
		}
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.HandleConnection(ctx, client1)

	crdtPayload := map[string]any{
		testKeyFileID: fileID.String(),
		testKeyDelta:  "Y3JkdC1kaWZm",
	}
	updateMsg, _ := websocket.NewMessage(websocket.MsgCRDTUpdate, crdtPayload)
	pushMessage(t, conn1, updateMsg)

	peerMsg := readOutgoingMessage(t, client2)
	if peerMsg.Type != websocket.MsgCRDTUpdate {
		t.Fatalf("expected peer to receive %s, got %s", websocket.MsgCRDTUpdate, peerMsg.Type)
	}

	var broadcastPayload crdtUpdatePayload
	if err := json.Unmarshal(peerMsg.Payload, &broadcastPayload); err != nil {
		t.Fatalf("failed to unmarshal broadcast crdt payload: %v", err)
	}
	if broadcastPayload.SenderID != client1.UserID() {
		t.Errorf("expected sender_id %s, got %s", client1.UserID(), broadcastPayload.SenderID)
	}
	if !applied {
		t.Error("expected applyFileChanges to be called")
	}

	cancel()
}

func TestService_HandleMetadataUpdate(t *testing.T) {
	t.Parallel()
	service, mockSync, room, client1, conn1 := setupTestSetup(t)

	client2 := websocket.NewClient(uuid.New(), "peer2@test.com", room.ProjectID(), newMockConnection(), room)
	room.Join(client2)
	<-client2.OutgoingMessages() // drain snapshot

	synced := false
	mockSync.syncFunc = func(
		_ context.Context,
		_ uuid.UUID,
		req *syncApp.Request,
	) (*syncApp.Response, error) {
		if string(req.MetadataDelta) == "meta-update-bytes" {
			synced = true
		}
		return &syncApp.Response{}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.HandleConnection(ctx, client1)

	metaPayload := map[string]any{
		testKeyDelta: "bWV0YS11cGRhdGUtYnl0ZXM=",
	}
	metaMsg, _ := websocket.NewMessage(websocket.MsgMetadataUpdate, metaPayload)
	pushMessage(t, conn1, metaMsg)

	peerMsg := readOutgoingMessage(t, client2)
	if peerMsg.Type != websocket.MsgMetadataUpdate {
		t.Fatalf("expected peer to receive %s, got %s", websocket.MsgMetadataUpdate, peerMsg.Type)
	}

	var broadcastPayload metadataUpdatePayload
	if err := json.Unmarshal(peerMsg.Payload, &broadcastPayload); err != nil {
		t.Fatalf("failed to unmarshal broadcast meta payload: %v", err)
	}
	if broadcastPayload.SenderID != client1.UserID() {
		t.Errorf("expected sender_id %s, got %s", client1.UserID(), broadcastPayload.SenderID)
	}
	if !synced {
		t.Error("expected syncService.Sync to be called for metadata update")
	}

	cancel()
}

func TestService_HandleAwarenessUpdate(t *testing.T) {
	t.Parallel()
	service, _, room, client1, conn1 := setupTestSetup(t)

	client2 := websocket.NewClient(uuid.New(), "peer3@test.com", room.ProjectID(), newMockConnection(), room)
	room.Join(client2)
	<-client2.OutgoingMessages() // drain snapshot

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.HandleConnection(ctx, client1)

	fileID := uuid.New()
	awarenessPayload := map[string]any{
		testKeyFileID: fileID.String(),
		"cursor": map[string]int{
			"anchor": 15,
			"head":   20,
		},
	}
	msg, _ := websocket.NewMessage(websocket.MsgAwarenessUpdate, awarenessPayload)
	pushMessage(t, conn1, msg)

	peerMsg := readOutgoingMessage(t, client2)
	if peerMsg.Type != websocket.MsgAwarenessUpdate {
		t.Fatalf("expected %s, got %s", websocket.MsgAwarenessUpdate, peerMsg.Type)
	}

	var presence websocket.UserPresence
	if err := json.Unmarshal(peerMsg.Payload, &presence); err != nil {
		t.Fatalf("failed to unmarshal user presence: %v", err)
	}
	if presence.UserID != client1.UserID() {
		t.Errorf("expected user_id %s, got %s", client1.UserID(), presence.UserID)
	}
	if presence.Cursor.Anchor != 15 || presence.Cursor.Head != 20 {
		t.Errorf("unexpected cursor: %+v", presence.Cursor)
	}

	cancel()
}

func TestService_Errors(t *testing.T) {
	t.Parallel()
	service, mockSync, _, client, mockConn := setupTestSetup(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.HandleConnection(ctx, client)

	// 1. Unknown message type
	unknownMsg, _ := websocket.NewMessage(websocket.MessageType("unknown_type"), nil)
	pushMessage(t, mockConn, unknownMsg)
	errMsg := readOutgoingMessage(t, client)
	if errMsg.Type != websocket.MsgError {
		t.Errorf("expected %s, got %s", websocket.MsgError, errMsg.Type)
	}

	// 2. Sync failure
	mockSync.syncFunc = func(
		_ context.Context,
		_ uuid.UUID,
		_ *syncApp.Request,
	) (*syncApp.Response, error) {
		return nil, errors.New("database failure")
	}
	initMsg, _ := websocket.NewMessage(websocket.MsgSyncInit, map[string]any{})
	pushMessage(t, mockConn, initMsg)
	syncErrMsg := readOutgoingMessage(t, client)
	if syncErrMsg.Type != websocket.MsgError {
		t.Errorf("expected %s, got %s", websocket.MsgError, syncErrMsg.Type)
	}

	cancel()
}
