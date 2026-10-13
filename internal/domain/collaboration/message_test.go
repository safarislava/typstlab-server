package collaboration

import "testing"

func TestMessageType_IsValid(t *testing.T) {
	t.Parallel()

	validTypes := []MessageType{
		MsgSyncInit,
		MsgSyncResponse,
		MsgCRDTUpdate,
		MsgMetadataUpdate,
		MsgAwarenessSnapshot,
		MsgAwarenessUpdate,
		MsgAwarenessRemove,
		MsgPing,
		MsgPong,
		MsgError,
	}

	for _, msgType := range validTypes {
		if !msgType.IsValid() {
			t.Errorf("expected %s to be valid", msgType)
		}
		if msgType.String() != string(msgType) {
			t.Errorf("expected String() to return %s, got %s", string(msgType), msgType.String())
		}
	}

	if MessageType("invalid_type").IsValid() {
		t.Error("expected invalid_type to be invalid")
	}
}
