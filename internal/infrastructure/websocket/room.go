package websocket

import (
	"sync"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"

	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
)

// Room manages active WebSocket connections and presence state for a single project.
type Room struct {
	projectID uuid.UUID
	awareness *AwarenessStore
	clients   map[*Client]bool
	mu        sync.RWMutex
}

// NewRoom creates a new Room for the given project.
func NewRoom(projectID uuid.UUID) *Room {
	return &Room{
		projectID: projectID,
		awareness: NewAwarenessStore(),
		clients:   make(map[*Client]bool),
	}
}

// ProjectID returns the project ID of this room.
func (r *Room) ProjectID() uuid.UUID {
	return r.projectID
}

// Awareness returns the in-memory AwarenessStore for this room.
func (r *Room) Awareness() *AwarenessStore {
	return r.awareness
}

// Join registers a client into the room and immediately sends an awareness snapshot.
func (r *Room) Join(client *Client) {
	r.mu.Lock()
	r.clients[client] = true
	snapshot := r.awareness.Snapshot()
	r.mu.Unlock()

	snapshotMsg, err := NewMessage(domainCollab.MsgAwarenessSnapshot, map[string]any{
		"presences": snapshot,
	})
	if err == nil {
		_ = client.Send(snapshotMsg)
	}
}

// Leave unregisters a client from the room and cleans up awareness if user is gone.
func (r *Room) Leave(client *Client) {
	r.mu.Lock()
	if !r.clients[client] {
		r.mu.Unlock()
		return
	}
	delete(r.clients, client)

	hasOtherConnections := false
	for c := range r.clients {
		if c.UserID() == client.UserID() {
			hasOtherConnections = true
			break
		}
	}
	r.mu.Unlock()

	if !hasOtherConnections {
		r.awareness.RemovePresence(client.UserID())
		removeMsg, err := NewMessage(domainCollab.MsgAwarenessRemove, map[string]any{
			"user_id": client.UserID(),
		})
		if err == nil {
			r.Broadcast(removeMsg)
		}
	}
}

// Broadcast sends a Message to all connected clients in the room.
func (r *Room) Broadcast(msg Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for c := range r.clients {
		_ = c.Send(msg)
	}
}

// BroadcastExcept sends a Message to all connected clients except the one with exceptID.
func (r *Room) BroadcastExcept(exceptID uuid.UUID, msg Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for c := range r.clients {
		if c.ID() == exceptID {
			continue
		}
		_ = c.Send(msg)
	}
}

// ClientsCount returns the current number of active connections in the room.
func (r *Room) ClientsCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Close closes all client connections in the room with the given status code.
func (r *Room) Close(status coderws.StatusCode, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for c := range r.clients {
		c.Close(status, reason)
	}
	r.clients = make(map[*Client]bool)
}
