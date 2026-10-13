package websocket

import (
	"encoding/json"
	"fmt"

	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
)

// Message is the WebSocket envelope carrying a typed message and its JSON payload.
type Message struct {
	Type    domainCollab.MessageType `json:"type"`
	Payload json.RawMessage          `json:"payload"`
}

// NewMessage constructs a Message envelope with JSON-encoded payload.
func NewMessage(msgType domainCollab.MessageType, payload any) (Message, error) {
	if payload == nil {
		return Message{
			Type:    msgType,
			Payload: json.RawMessage("null"),
		}, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("marshal message payload: %w", err)
	}
	return Message{
		Type:    msgType,
		Payload: raw,
	}, nil
}
