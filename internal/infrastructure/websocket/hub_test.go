package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHub_GetOrCreateRoom(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	projectID := uuid.New()

	room1 := hub.GetOrCreateRoom(projectID)
	if room1 == nil {
		t.Fatal("Expected room to be created")
	}
	if room1.ProjectID() != projectID {
		t.Errorf("Expected projectID %v, got %v", projectID, room1.ProjectID())
	}

	room2 := hub.GetOrCreateRoom(projectID)
	if room1 != room2 {
		t.Error("Expected GetOrCreateRoom to return same room instance")
	}

	if hub.RoomsCount() != 1 {
		t.Errorf("Expected RoomsCount 1, got %d", hub.RoomsCount())
	}

	retrieved, exists := hub.GetRoom(projectID)
	if !exists || retrieved != room1 {
		t.Errorf("Expected GetRoom to return room1, got %v (exists: %v)", retrieved, exists)
	}
}

func TestHub_RemoveClientAndEmptyRoom(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	projectID := uuid.New()
	room := hub.GetOrCreateRoom(projectID)

	userID := uuid.New()
	client := NewClient(userID, "test@test.com", projectID, nil, room)

	room.Join(client)
	if room.ClientsCount() != 1 {
		t.Errorf("Expected 1 client in room, got %d", room.ClientsCount())
	}

	// Joining client should have received MsgAwarenessSnapshot in sendChan
	select {
	case data := <-client.sendChan:
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("Failed to decode message: %v", err)
		}
		if msg.Type != MsgAwarenessSnapshot {
			t.Errorf("Expected message type %q, got %q", MsgAwarenessSnapshot, msg.Type)
		}
	default:
		t.Error("Expected client to receive awareness snapshot on join")
	}

	hub.RemoveClient(client)
	if room.ClientsCount() != 0 {
		t.Errorf("Expected 0 clients in room, got %d", room.ClientsCount())
	}

	// Room should be cleaned up from hub
	if hub.RoomsCount() != 0 {
		t.Errorf("Expected empty room to be removed from Hub, got %d rooms", hub.RoomsCount())
	}
}

func TestRoom_BroadcastAndBroadcastExcept(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	projectID := uuid.New()
	room := hub.GetOrCreateRoom(projectID)

	client1 := NewClient(uuid.New(), "u1@test.com", projectID, nil, room)
	client2 := NewClient(uuid.New(), "u2@test.com", projectID, nil, room)

	room.Join(client1)
	room.Join(client2)

	// Drain snapshot messages
	<-client1.sendChan
	<-client2.sendChan

	testMsg, err := NewMessage(MsgCRDTUpdate, map[string]string{"sample": "data"})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// 1. Test BroadcastExcept: only client2 should receive it
	room.BroadcastExcept(client1.ID(), testMsg)

	select {
	case data := <-client2.sendChan:
		var msg Message
		_ = json.Unmarshal(data, &msg)
		if msg.Type != MsgCRDTUpdate {
			t.Errorf("Expected MsgCRDTUpdate, got %q", msg.Type)
		}
	default:
		t.Error("client2 should have received BroadcastExcept message")
	}

	select {
	case <-client1.sendChan:
		t.Error("client1 should NOT receive BroadcastExcept message")
	default:
		// expected
	}

	// 2. Test Broadcast: both clients should receive it
	room.Broadcast(testMsg)

	select {
	case <-client1.sendChan:
		// expected
	default:
		t.Error("client1 should have received Broadcast message")
	}

	select {
	case <-client2.sendChan:
		// expected
	default:
		t.Error("client2 should have received Broadcast message")
	}
}

func TestClient_AccessorsAndSend(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	userID := uuid.New()
	userEmail := "u@test.com"
	room := NewRoom(projectID)
	client := NewClient(userID, userEmail, projectID, nil, room)

	if client.UserID() != userID {
		t.Errorf("Expected userID %v, got %v", userID, client.UserID())
	}
	if client.UserEmail() != userEmail {
		t.Errorf("Expected userEmail %v, got %v", userEmail, client.UserEmail())
	}
	if client.ProjectID() != projectID {
		t.Errorf("Expected projectID %v, got %v", projectID, client.ProjectID())
	}
	if client.Room() != room {
		t.Errorf("Expected room %v, got %v", room, client.Room())
	}
	if client.ID() == uuid.Nil {
		t.Error("Expected client ID to not be nil UUID")
	}

	testMsg, err := NewMessage(MsgCRDTUpdate, map[string]string{"test": "123"})
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	if sendErr := client.Send(testMsg); sendErr != nil {
		t.Fatalf("Failed to send message: %v", sendErr)
	}

	client.Close(0, "test close")
	if sendErr := client.Send(testMsg); sendErr == nil {
		t.Error("Expected Send to fail after client context is closed")
	}
}

func TestHub_Shutdown(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	p1 := uuid.New()
	p2 := uuid.New()

	_ = hub.GetOrCreateRoom(p1)
	_ = hub.GetOrCreateRoom(p2)

	if hub.RoomsCount() != 2 {
		t.Fatalf("Expected 2 rooms, got %d", hub.RoomsCount())
	}

	hub.Shutdown(context.Background())

	if hub.RoomsCount() != 0 {
		t.Errorf("Expected 0 rooms after Shutdown, got %d", hub.RoomsCount())
	}
}

func TestUpgrader_InvalidRequest(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"*"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)

	_, err := upgrader.Upgrade(rec, req)
	if err == nil {
		t.Fatal("Expected upgrade to fail for regular HTTP request without WebSocket headers")
	}
}
