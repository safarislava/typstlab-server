package websocket

import (
	"encoding/binary"
	"sync"

	"github.com/google/uuid"
)

var defaultColorPalette = []string{
	"#3B82F6", // Blue
	"#10B981", // Emerald
	"#F59E0B", // Amber
	"#EF4444", // Red
	"#8B5CF6", // Violet
	"#EC4899", // Pink
	"#06B6D4", // Cyan
	"#84CC16", // Lime
	"#F97316", // Orange
	"#6366F1", // Indigo
	"#14B8A6", // Teal
	"#D946EF", // Fuchsia
}

// Cursor represents the caret position and text selection in an active file.
type Cursor struct {
	Anchor int `json:"anchor"`
	Head   int `json:"head"`
}

// UserPresence represents the active presence and cursor of a collaborator.
type UserPresence struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Color  string    `json:"color"`
	FileID uuid.UUID `json:"file_id"`
	Cursor Cursor    `json:"cursor"`
}

// ColorForUser returns a deterministic color from the palette for a given user ID.
func ColorForUser(id uuid.UUID) string {
	val := binary.BigEndian.Uint32(id[:4])
	return defaultColorPalette[val%uint32(len(defaultColorPalette))]
}

// AwarenessStore is a thread-safe in-memory store for active collaborator presences and cursors in a room.
type AwarenessStore struct {
	mu        sync.RWMutex
	presences map[uuid.UUID]UserPresence
}

// NewAwarenessStore creates an empty AwarenessStore.
func NewAwarenessStore() *AwarenessStore {
	return &AwarenessStore{
		presences: make(map[uuid.UUID]UserPresence),
	}
}

// SetPresence updates or inserts a user's presence and cursor.
func (s *AwarenessStore) SetPresence(userID uuid.UUID, email string, fileID uuid.UUID, cursor Cursor) UserPresence {
	s.mu.Lock()
	defer s.mu.Unlock()

	presence := UserPresence{
		UserID: userID,
		Email:  email,
		Color:  ColorForUser(userID),
		FileID: fileID,
		Cursor: cursor,
	}

	s.presences[userID] = presence
	return presence
}

// RemovePresence removes a user's presence from the store. Returns true if user was present.
func (s *AwarenessStore) RemovePresence(userID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.presences[userID]; exists {
		delete(s.presences, userID)
		return true
	}
	return false
}

// GetPresence retrieves a copy of the user's presence if present.
func (s *AwarenessStore) GetPresence(userID uuid.UUID) (UserPresence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.presences[userID]
	return p, exists
}

// Snapshot returns a point-in-time snapshot of all active user presences in the store.
func (s *AwarenessStore) Snapshot() []UserPresence {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]UserPresence, 0, len(s.presences))
	for _, p := range s.presences {
		list = append(list, p)
	}
	return list
}

// Count returns the number of active presences.
func (s *AwarenessStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.presences)
}
