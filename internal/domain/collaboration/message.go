package collaboration

// MessageType represents the type of a real-time collaboration message.
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

// String returns the string representation of MessageType.
func (m MessageType) String() string {
	return string(m)
}

// IsValid checks whether the MessageType is a recognized collaboration message type.
func (m MessageType) IsValid() bool {
	switch m {
	case MsgSyncInit,
		MsgSyncResponse,
		MsgCRDTUpdate,
		MsgMetadataUpdate,
		MsgAwarenessSnapshot,
		MsgAwarenessUpdate,
		MsgAwarenessRemove,
		MsgPing,
		MsgPong,
		MsgError:
		return true
	default:
		return false
	}
}
