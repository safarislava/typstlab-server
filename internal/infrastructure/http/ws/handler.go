package ws

import (
	"context"
	"net/http"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"

	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
	"github.com/safarislava/typstlab-server/internal/infrastructure/websocket"
)

// UserService defines the contract for fetching user details.
type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

// CollaborationService defines the contract for running the collaboration loop.
type CollaborationService interface {
	HandleConnection(ctx context.Context, client *websocket.Client)
}

// Upgrader defines the contract for upgrading HTTP connections to WebSocket.
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request) (websocket.Connection, error)
}

// Handler handles WebSocket connection establishment and handshakes.
type Handler struct {
	upgrader             Upgrader
	hub                  *websocket.Hub
	userService          UserService
	collaborationService CollaborationService
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(
	upgrader Upgrader,
	hub *websocket.Hub,
	userService UserService,
	collaborationService CollaborationService,
) *Handler {
	return &Handler{
		upgrader:             upgrader,
		hub:                  hub,
		userService:          userService,
		collaborationService: collaborationService,
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

	h.collaborationService.HandleConnection(r.Context(), client)
}
