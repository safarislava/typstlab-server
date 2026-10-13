package websocket

import (
	"encoding/json"
	"testing"

	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
)

const (
	testHello = "hello"
)

func TestNewMessage_NilPayload(t *testing.T) {
	t.Parallel()

	msg, err := NewMessage(domainCollab.MsgPing, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg.Type != domainCollab.MsgPing {
		t.Errorf("Expected type %q, got %q", domainCollab.MsgPing, msg.Type)
	}
	if string(msg.Payload) != "null" {
		t.Errorf("Expected payload 'null', got %s", msg.Payload)
	}
}

func TestNewMessage_WithPayload(t *testing.T) {
	t.Parallel()

	type samplePayload struct {
		Text string `json:"text"`
	}

	payload := samplePayload{Text: testHello}
	msg, err := NewMessage(domainCollab.MsgCRDTUpdate, payload)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg.Type != domainCollab.MsgCRDTUpdate {
		t.Errorf("Expected type %q, got %q", domainCollab.MsgCRDTUpdate, msg.Type)
	}

	var decoded samplePayload
	if unmarshalErr := json.Unmarshal(msg.Payload, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal payload: %v", unmarshalErr)
	}
	if decoded.Text != testHello {
		t.Errorf("Expected text %q, got %s", testHello, decoded.Text)
	}
}

func TestNewMessage_InvalidPayload(t *testing.T) {
	t.Parallel()

	// Channel cannot be JSON-marshaled
	ch := make(chan int)
	_, err := NewMessage(domainCollab.MsgError, ch)
	if err == nil {
		t.Fatal("Expected error for non-serializable payload")
	}
}
