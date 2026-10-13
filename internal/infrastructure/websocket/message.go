package websocket

import (
	"encoding/json"
	"fmt"

	domainCollab "github.com/safarislava/typstlab-server/internal/domain/collaboration"
)

// MessageType aliases the domain collaboration message type.
type MessageType = domainCollab.MessageType

const (
	MsgSyncInit          = domainCollab.MsgSyncInit
	MsgSyncResponse      = domainCollab.MsgSyncResponse
	MsgCRDTUpdate        = domainCollab.MsgCRDTUpdate
	MsgMetadataUpdate    = domainCollab.MsgMetadataUpdate
	MsgAwarenessSnapshot = domainCollab.MsgAwarenessSnapshot
	MsgAwarenessUpdate   = domainCollab.MsgAwarenessUpdate
	MsgAwarenessRemove   = domainCollab.MsgAwarenessRemove
	MsgPing              = domainCollab.MsgPing
	MsgPong              = domainCollab.MsgPong
	MsgError             = domainCollab.MsgError
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
