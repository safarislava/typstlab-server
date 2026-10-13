package collaboration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	"github.com/safarislava/typstlab-server/internal/infrastructure/websocket"
)

// SyncService defines the contract required for synchronizing project files and metadata.
type SyncService interface {
	Sync(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error)
	ApplyFileChanges(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
}

type messageHandler func(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error

// Service orchestrates real-time document synchronization and awareness over WebSocket connections.
type Service struct {
	syncService SyncService
	handlers    map[domainCollab.MessageType]messageHandler
}

// NewService creates a new collaboration Service.
func NewService(syncService SyncService) *Service {
	s := &Service{
		syncService: syncService,
	}
	s.registerHandlers()
	return s
}

func (s *Service) registerHandlers() {
	s.handlers = map[domainCollab.MessageType]messageHandler{
		domainCollab.MsgPing:            s.handlePing,
		domainCollab.MsgPong:            s.handlePong,
		domainCollab.MsgSyncInit:        s.handleSyncInit,
		domainCollab.MsgCRDTUpdate:      s.handleCRDTUpdate,
		domainCollab.MsgMetadataUpdate:  s.handleMetadataUpdate,
		domainCollab.MsgAwarenessUpdate: s.handleAwarenessUpdate,
	}
}

// HandleConnection processes incoming WebSocket messages for the client until disconnection.
func (s *Service) HandleConnection(ctx context.Context, client *websocket.Client) {
	for {
		msg, err := client.ReadMessage()
		if err != nil {
			return
		}

		if dispatchErr := s.dispatchMessage(ctx, client, msg); dispatchErr != nil {
			s.sendErrorMessage(client, dispatchErr.Error())
		}
	}
}

func (s *Service) dispatchMessage(ctx context.Context, client *websocket.Client, msg websocket.Message) error {
	handler, exists := s.handlers[msg.Type]
	if !exists {
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
	return handler(ctx, client, msg.Payload)
}

func (s *Service) handlePing(_ context.Context, client *websocket.Client, _ json.RawMessage) error {
	pongMsg, err := websocket.NewMessage(websocket.MsgPong, nil)
	if err != nil {
		return fmt.Errorf("create pong message: %w", err)
	}
	if sendErr := client.Send(pongMsg); sendErr != nil {
		return fmt.Errorf("send pong message: %w", sendErr)
	}
	return nil
}

func (s *Service) handlePong(_ context.Context, _ *websocket.Client, _ json.RawMessage) error {
	return nil
}

type syncInitPayload struct {
	MetadataDelta       []byte               `json:"metadata_delta"`
	MetadataStateVector []byte               `json:"metadata_state_vector"`
	ContentVectors      map[uuid.UUID][]byte `json:"content_vectors"`
}

type syncInstructionPayload struct {
	Action string    `json:"action"`
	FileID uuid.UUID `json:"file_id"`
	Delta  []byte    `json:"delta"`
}

type syncResponsePayload struct {
	MetadataDelta []byte                   `json:"metadata_delta"`
	Instructions  []syncInstructionPayload `json:"instructions"`
}

func (s *Service) handleSyncInit(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error {
	var payload syncInitPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal sync init payload: %w", err)
	}

	appReq := &syncApp.Request{
		MetadataDelta:       payload.MetadataDelta,
		MetadataStateVector: payload.MetadataStateVector,
		ContentVectors:      payload.ContentVectors,
	}

	resp, err := s.syncService.Sync(ctx, client.ProjectID(), appReq)
	if err != nil {
		return fmt.Errorf("execute sync init: %w", err)
	}

	return s.sendSyncResponse(client, resp)
}

func (s *Service) sendSyncResponse(client *websocket.Client, resp *syncApp.Response) error {
	instructions := make([]syncInstructionPayload, 0, len(resp.Instructions))
	for _, inst := range resp.Instructions {
		instructions = append(instructions, syncInstructionPayload{
			Action: string(inst.Action),
			FileID: inst.FileID,
			Delta:  inst.Delta,
		})
	}

	outPayload := syncResponsePayload{
		MetadataDelta: resp.MetadataDelta,
		Instructions:  instructions,
	}

	msg, err := websocket.NewMessage(websocket.MsgSyncResponse, outPayload)
	if err != nil {
		return fmt.Errorf("create sync response message: %w", err)
	}

	if sendErr := client.Send(msg); sendErr != nil {
		return fmt.Errorf("send sync response: %w", sendErr)
	}
	return nil
}

type crdtUpdatePayload struct {
	FileID   uuid.UUID `json:"file_id"`
	SenderID uuid.UUID `json:"sender_id"`
	Delta    []byte    `json:"delta"`
}

func (s *Service) handleCRDTUpdate(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error {
	var payload crdtUpdatePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal crdt update payload: %w", err)
	}

	if payload.FileID == uuid.Nil {
		return errors.New("file_id is required in crdt update")
	}
	if len(payload.Delta) == 0 {
		return errors.New("delta is empty in crdt update")
	}

	req := syncApp.ApplyFileChangesRequest{
		FileID: payload.FileID,
		Delta:  payload.Delta,
	}
	if _, err := s.syncService.ApplyFileChanges(ctx, req); err != nil {
		return fmt.Errorf("apply crdt changes: %w", err)
	}

	return s.broadcastCRDTUpdate(client, payload)
}

func (s *Service) broadcastCRDTUpdate(client *websocket.Client, payload crdtUpdatePayload) error {
	broadcastPayload := crdtUpdatePayload{
		FileID:   payload.FileID,
		SenderID: client.UserID(),
		Delta:    payload.Delta,
	}

	msg, err := websocket.NewMessage(websocket.MsgCRDTUpdate, broadcastPayload)
	if err != nil {
		return fmt.Errorf("create crdt broadcast message: %w", err)
	}

	if room := client.Room(); room != nil {
		room.BroadcastExcept(client.ID(), msg)
	}
	return nil
}

type metadataUpdatePayload struct {
	SenderID uuid.UUID `json:"sender_id"`
	Delta    []byte    `json:"delta"`
}

func (s *Service) handleMetadataUpdate(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error {
	var payload metadataUpdatePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal metadata update payload: %w", err)
	}

	if len(payload.Delta) == 0 {
		return errors.New("delta is empty in metadata update")
	}

	req := &syncApp.Request{
		MetadataDelta: payload.Delta,
	}
	if _, err := s.syncService.Sync(ctx, client.ProjectID(), req); err != nil {
		return fmt.Errorf("sync metadata update: %w", err)
	}

	return s.broadcastMetadataUpdate(client, payload.Delta)
}

func (s *Service) broadcastMetadataUpdate(client *websocket.Client, delta []byte) error {
	broadcastPayload := metadataUpdatePayload{
		SenderID: client.UserID(),
		Delta:    delta,
	}

	msg, err := websocket.NewMessage(websocket.MsgMetadataUpdate, broadcastPayload)
	if err != nil {
		return fmt.Errorf("create metadata broadcast message: %w", err)
	}

	if room := client.Room(); room != nil {
		room.BroadcastExcept(client.ID(), msg)
	}
	return nil
}

type awarenessUpdatePayload struct {
	FileID uuid.UUID        `json:"file_id"`
	Cursor websocket.Cursor `json:"cursor"`
}

func (s *Service) handleAwarenessUpdate(
	_ context.Context,
	client *websocket.Client,
	rawPayload json.RawMessage,
) error {
	var payload awarenessUpdatePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal awareness update payload: %w", err)
	}

	room := client.Room()
	if room == nil {
		return errors.New("client has no room for awareness update")
	}

	presence := room.Awareness().SetPresence(
		client.UserID(),
		client.UserEmail(),
		payload.FileID,
		payload.Cursor,
	)

	msg, err := websocket.NewMessage(websocket.MsgAwarenessUpdate, presence)
	if err != nil {
		return fmt.Errorf("create awareness broadcast message: %w", err)
	}

	room.BroadcastExcept(client.ID(), msg)
	return nil
}

type errorPayload struct {
	Message string `json:"message"`
}

func (s *Service) sendErrorMessage(client *websocket.Client, errMsg string) {
	msg, err := websocket.NewMessage(websocket.MsgError, errorPayload{
		Message: errMsg,
	})
	if err == nil {
		_ = client.Send(msg)
	}
}
