package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"

	collabApp "github.com/safarislava/typstlab-server/internal/application/collaboration"
	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
	"github.com/safarislava/typstlab-server/internal/infrastructure/websocket"
)

// UserService defines the contract for fetching user details.
type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

// CollaborationService defines the contract for document synchronization operations.
type CollaborationService interface {
	SyncInit(ctx context.Context, projectID uuid.UUID, req collabApp.SyncInitRequest) (*collabApp.SyncInitResponse, error)
	ApplyCRDTUpdate(ctx context.Context, fileID uuid.UUID, delta []byte) error
	ApplyMetadataUpdate(ctx context.Context, projectID uuid.UUID, delta []byte) error
}

// Upgrader defines the contract for upgrading HTTP connections to WebSocket.
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request) (websocket.Connection, error)
}

type messageHandler func(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error

// Handler handles WebSocket connection establishment, message dispatching, and collaboration.
type Handler struct {
	upgrader             Upgrader
	hub                  *websocket.Hub
	userService          UserService
	collaborationService CollaborationService
	handlers             map[domainCollab.MessageType]messageHandler
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(
	upgrader Upgrader,
	hub *websocket.Hub,
	userService UserService,
	collaborationService CollaborationService,
) *Handler {
	h := &Handler{
		upgrader:             upgrader,
		hub:                  hub,
		userService:          userService,
		collaborationService: collaborationService,
	}
	h.registerHandlers()
	return h
}

func (h *Handler) registerHandlers() {
	h.handlers = map[domainCollab.MessageType]messageHandler{
		domainCollab.MsgPing:            h.handlePing,
		domainCollab.MsgPong:            h.handlePong,
		domainCollab.MsgSyncInit:        h.handleSyncInit,
		domainCollab.MsgCRDTUpdate:      h.handleCRDTUpdate,
		domainCollab.MsgMetadataUpdate:  h.handleMetadataUpdate,
		domainCollab.MsgAwarenessUpdate: h.handleAwarenessUpdate,
	}
}

// ServeWS upgrades the HTTP connection to WebSocket, creates a client session, and handles collaboration.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r)
	if err != nil {
		return
	}

	room := h.hub.GetOrCreateRoom(project.ID())
	client := websocket.NewClient(user.ID(), user.Email(), project.ID(), conn, room)
	room.Join(client)

	go client.WritePump()

	defer func() {
		h.hub.RemoveClient(client)
		client.Close(coderws.StatusNormalClosure, "connection closed")
	}()

	h.handleConnection(r.Context(), client)
}

func (h *Handler) handleConnection(ctx context.Context, client *websocket.Client) {
	for {
		msg, err := client.ReadMessage()
		if err != nil {
			return
		}

		if dispatchErr := h.dispatchMessage(ctx, client, msg); dispatchErr != nil {
			h.sendErrorMessage(client, dispatchErr.Error())
		}
	}
}

func (h *Handler) dispatchMessage(ctx context.Context, client *websocket.Client, msg websocket.Message) error {
	handler, exists := h.handlers[msg.Type]
	if !exists {
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
	return handler(ctx, client, msg.Payload)
}

func (h *Handler) handlePing(_ context.Context, client *websocket.Client, _ json.RawMessage) error {
	pongMsg, err := websocket.NewMessage(domainCollab.MsgPong, nil)
	if err != nil {
		return fmt.Errorf("create pong message: %w", err)
	}
	if sendErr := client.Send(pongMsg); sendErr != nil {
		return fmt.Errorf("send pong message: %w", sendErr)
	}
	return nil
}

func (h *Handler) handlePong(_ context.Context, _ *websocket.Client, _ json.RawMessage) error {
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

func (h *Handler) handleSyncInit(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error {
	var payload syncInitPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal sync init payload: %w", err)
	}

	appReq := collabApp.SyncInitRequest{
		MetadataDelta:       payload.MetadataDelta,
		MetadataStateVector: payload.MetadataStateVector,
		ContentVectors:      payload.ContentVectors,
	}

	resp, err := h.collaborationService.SyncInit(ctx, client.ProjectID(), appReq)
	if err != nil {
		return fmt.Errorf("execute sync init: %w", err)
	}

	instructions := make([]syncInstructionPayload, 0, len(resp.Instructions))
	for _, inst := range resp.Instructions {
		instructions = append(instructions, syncInstructionPayload{
			Action: inst.Action,
			FileID: inst.FileID,
			Delta:  inst.Delta,
		})
	}

	outPayload := syncResponsePayload{
		MetadataDelta: resp.MetadataDelta,
		Instructions:  instructions,
	}

	msg, err := websocket.NewMessage(domainCollab.MsgSyncResponse, outPayload)
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

func (h *Handler) handleCRDTUpdate(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error {
	var payload crdtUpdatePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal crdt update payload: %w", err)
	}

	if err := h.collaborationService.ApplyCRDTUpdate(ctx, payload.FileID, payload.Delta); err != nil {
		return fmt.Errorf("apply crdt changes: %w", err)
	}

	broadcastPayload := crdtUpdatePayload{
		FileID:   payload.FileID,
		SenderID: client.UserID(),
		Delta:    payload.Delta,
	}

	msg, err := websocket.NewMessage(domainCollab.MsgCRDTUpdate, broadcastPayload)
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

func (h *Handler) handleMetadataUpdate(ctx context.Context, client *websocket.Client, rawPayload json.RawMessage) error {
	var payload metadataUpdatePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return fmt.Errorf("unmarshal metadata update payload: %w", err)
	}

	if err := h.collaborationService.ApplyMetadataUpdate(ctx, client.ProjectID(), payload.Delta); err != nil {
		return fmt.Errorf("apply metadata update: %w", err)
	}

	broadcastPayload := metadataUpdatePayload{
		SenderID: client.UserID(),
		Delta:    payload.Delta,
	}

	msg, err := websocket.NewMessage(domainCollab.MsgMetadataUpdate, broadcastPayload)
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

func (h *Handler) handleAwarenessUpdate(
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

	msg, err := websocket.NewMessage(domainCollab.MsgAwarenessUpdate, presence)
	if err != nil {
		return fmt.Errorf("create awareness broadcast message: %w", err)
	}

	room.BroadcastExcept(client.ID(), msg)
	return nil
}

type errorPayload struct {
	Message string `json:"message"`
}

func (h *Handler) sendErrorMessage(client *websocket.Client, errMsg string) {
	msg, err := websocket.NewMessage(domainCollab.MsgError, errorPayload{
		Message: errMsg,
	})
	if err == nil {
		_ = client.Send(msg)
	}
}
