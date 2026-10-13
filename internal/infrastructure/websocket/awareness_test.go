package websocket

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
)

const (
	testAliceEmail = "alice@example.com"
)

func TestColorForUser_Deterministic(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	color1 := ColorForUser(id)
	color2 := ColorForUser(id)

	if color1 != color2 {
		t.Fatalf("Expected deterministic color %q, got %q", color1, color2)
	}

	foundInPalette := false
	for _, c := range defaultColorPalette {
		if c == color1 {
			foundInPalette = true
			break
		}
	}
	if !foundInPalette {
		t.Errorf("Color %q not found in defaultColorPalette", color1)
	}
}

func TestAwarenessStore_SetAndGetPresence(t *testing.T) {
	t.Parallel()

	store := NewAwarenessStore()
	userID := uuid.New()
	fileID := uuid.New()
	cursor := Cursor{Anchor: 10, Head: 25}

	created := store.SetPresence(userID, testAliceEmail, fileID, cursor)
	assertPresenceFields(t, &created, userID, testAliceEmail, fileID)

	got, ok := store.GetPresence(userID)
	if !ok {
		t.Fatal("Expected presence to be found")
	}
	if got.Cursor.Anchor != 10 {
		t.Errorf("Expected cursor anchor 10, got %d", got.Cursor.Anchor)
	}
}

func TestAwarenessStore_UpdatePresence(t *testing.T) {
	t.Parallel()

	store := NewAwarenessStore()
	userID := uuid.New()
	fileID := uuid.New()

	store.SetPresence(userID, testAliceEmail, fileID, Cursor{Anchor: 10, Head: 25})

	newCursor := Cursor{Anchor: 50, Head: 50}
	updated := store.SetPresence(userID, testAliceEmail, fileID, newCursor)
	if updated.Cursor.Anchor != 50 {
		t.Errorf("Expected updated cursor anchor 50, got %+v", updated.Cursor)
	}

	if store.Count() != 1 {
		t.Errorf("Expected count 1, got %d", store.Count())
	}
}

func assertPresenceFields(t *testing.T, p *UserPresence, userID uuid.UUID, email string, fileID uuid.UUID) {
	t.Helper()

	if p.UserID != userID || p.Email != email {
		t.Errorf("Unexpected presence user or email: %+v", p)
	}
	if p.FileID != fileID {
		t.Errorf("Expected fileID %v, got %v", fileID, p.FileID)
	}
	if p.Cursor.Anchor != 10 || p.Cursor.Head != 25 {
		t.Errorf("Expected cursor {10, 25}, got %+v", p.Cursor)
	}
}

func TestAwarenessStore_RemovePresence(t *testing.T) {
	t.Parallel()

	store := NewAwarenessStore()
	userID := uuid.New()

	store.SetPresence(userID, "bob@example.com", uuid.Nil, Cursor{})
	if store.Count() != 1 {
		t.Fatalf("Expected count 1, got %d", store.Count())
	}

	removed := store.RemovePresence(userID)
	if !removed {
		t.Error("Expected RemovePresence to return true for existing user")
	}
	if store.Count() != 0 {
		t.Errorf("Expected count 0, got %d", store.Count())
	}

	removedAgain := store.RemovePresence(userID)
	if removedAgain {
		t.Error("Expected RemovePresence to return false for non-existing user")
	}
}

func TestAwarenessStore_Snapshot(t *testing.T) {
	t.Parallel()

	store := NewAwarenessStore()
	user1 := uuid.New()
	user2 := uuid.New()

	store.SetPresence(user1, "u1@test.com", uuid.Nil, Cursor{Anchor: 1, Head: 1})
	store.SetPresence(user2, "u2@test.com", uuid.Nil, Cursor{Anchor: 2, Head: 2})

	snapshot := store.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("Expected snapshot length 2, got %d", len(snapshot))
	}

	// Mutating the store after snapshot should not affect the snapshot
	user3 := uuid.New()
	store.SetPresence(user3, "u3@test.com", uuid.Nil, Cursor{})

	if len(snapshot) != 2 {
		t.Errorf("Snapshot slice should not be modified after store mutation")
	}
	if store.Count() != 3 {
		t.Errorf("Expected store count 3, got %d", store.Count())
	}
}

func TestAwarenessStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := NewAwarenessStore()
	const numGoroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()
			user := uuid.New()
			email := fmt.Sprintf("worker-%d@test.com", workerID)

			for j := 0; j < iterations; j++ {
				store.SetPresence(user, email, uuid.Nil, Cursor{Anchor: j, Head: j})
				_ = store.Snapshot()
				_, _ = store.GetPresence(user)
				_ = store.Count()
			}

			store.RemovePresence(user)
		}(i)
	}

	wg.Wait()

	if store.Count() != 0 {
		t.Errorf("Expected store to be empty after all workers cleaned up, got %d", store.Count())
	}
}
