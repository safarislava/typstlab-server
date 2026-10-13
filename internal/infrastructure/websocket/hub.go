package websocket

import (
	"context"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// Hub is the global registry managing project rooms and active connections.
type Hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*Room
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[uuid.UUID]*Room),
	}
}

// GetOrCreateRoom returns an existing Room for the project or creates and registers a new one.
func (h *Hub) GetOrCreateRoom(projectID uuid.UUID) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, exists := h.rooms[projectID]; exists {
		return room
	}

	room := NewRoom(projectID)
	h.rooms[projectID] = room
	return room
}

// GetRoom retrieves a room by project ID if it exists.
func (h *Hub) GetRoom(projectID uuid.UUID) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[projectID]
	return room, exists
}

// RemoveClient unregisters a client from its room and removes the room from Hub if it becomes empty.
func (h *Hub) RemoveClient(client *Client) {
	if client == nil {
		return
	}
	room := client.Room()
	if room == nil {
		return
	}

	room.Leave(client)

	h.mu.Lock()
	defer h.mu.Unlock()
	if room.ClientsCount() == 0 {
		delete(h.rooms, room.ProjectID())
	}
}

// RoomsCount returns the current count of active project rooms.
func (h *Hub) RoomsCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// Shutdown closes all active rooms and their client connections gracefully.
func (h *Hub) Shutdown(_ context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range h.rooms {
		room.Close(websocket.StatusGoingAway, "server shutting down")
	}
	h.rooms = make(map[uuid.UUID]*Room)
}
