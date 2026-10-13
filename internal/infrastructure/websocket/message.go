package websocket

import (
	"encoding/json"
	"fmt"
)

type MessageType string

const (
	MsgSyncInit          MessageType = "sync_init"
	MsgSyncResponse      MessageType = "sync_response"
	MsgCRDTUpdate        MessageType = "crdt_update"
	MsgMetadataUpdate    MessageType = "metadata_update"
	MsgAwarenessSnapshot MessageType = "awareness_snapshot"
	MsgAwarenessUpdate   MessageType = "awareness_update"
	MsgAwarenessRemove   MessageType = "awareness_remove"
	MsgPing              MessageType = "ping"
	MsgPong              MessageType = "pong"
	MsgError             MessageType = "error"
)

// Message is the WebSocket envelope carrying a typed message and its JSON payload.
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewMessage constructs a Message envelope with JSON-encoded payload.
func NewMessage(msgType MessageType, payload any) (Message, error) {
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
