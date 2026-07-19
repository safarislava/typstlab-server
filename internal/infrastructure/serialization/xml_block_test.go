package serialization

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

const (
	testBlockName    = "Введение"
	testBlockContent = "= Введение\nТекст"
)

var testBlockID1 = uuid.MustParse("10000000-0000-0000-0000-000000000001")

func TestSerializeBlock(t *testing.T) {
	t.Parallel()

	state := []byte("state-bytes")
	b, _ := block.NewBlock(testBlockID1, testBlockName, state, testBlockContent)

	xb := serializeBlock(b)

	if xb.ID != testBlockID1.String() {
		t.Errorf("Expected ID %s, got %s", testBlockID1, xb.ID)
	}
	if xb.Name != testBlockName {
		t.Errorf("Expected Name 'Введение', got %q", xb.Name)
	}
	expectedState := base64.StdEncoding.EncodeToString(state)
	if xb.State != expectedState {
		t.Errorf("Expected State %s, got %s", expectedState, xb.State)
	}
	if xb.Content != testBlockContent {
		t.Errorf("Expected Content '= Введение\\nТекст', got %q", xb.Content)
	}
}

func TestDeserializeBlock(t *testing.T) {
	t.Parallel()

	state := []byte("state-bytes")
	stateB64 := base64.StdEncoding.EncodeToString(state)

	xb := xmlBlock{
		ID:      testBlockID1.String(),
		Name:    testBlockName,
		State:   stateB64,
		Content: testBlockContent,
	}

	b, err := deserializeBlock(&xb)
	if err != nil {
		t.Fatalf("deserializeBlock failed: %v", err)
	}

	if b.ID() != testBlockID1 {
		t.Errorf("Expected ID %s, got %s", testBlockID1, b.ID())
	}
	if b.Name() != testBlockName {
		t.Errorf("Expected Name 'Введение', got %q", b.Name())
	}
	if !bytes.Equal(b.State(), state) {
		t.Errorf("State mismatch")
	}
	if b.Content() != testBlockContent {
		t.Errorf("Expected Content '= Введение\\nТекст', got %q", b.Content())
	}
}

func TestSerializeDeserializeBlock_Roundtrip(t *testing.T) {
	t.Parallel()

	state := []byte{0x00, 0xFF, 0xAB, 0xCD}
	original, _ := block.NewBlock(testBlockID1, "Chapter", state, "= Chapter\nContent")

	xb := serializeBlock(original)
	restored, err := deserializeBlock(&xb)
	if err != nil {
		t.Fatalf("deserializeBlock failed: %v", err)
	}

	if original.ID() != restored.ID() {
		t.Errorf("ID mismatch: %s vs %s", original.ID(), restored.ID())
	}
	if original.Name() != restored.Name() {
		t.Errorf("Name mismatch: %q vs %q", original.Name(), restored.Name())
	}
	if !bytes.Equal(original.State(), restored.State()) {
		t.Errorf("State mismatch")
	}
	if original.Content() != restored.Content() {
		t.Errorf("Content mismatch: %q vs %q", original.Content(), restored.Content())
	}
}

func TestDeserializeBlock_InvalidID(t *testing.T) {
	t.Parallel()

	xb := xmlBlock{ID: "not-a-uuid", Name: "T", State: "dGVzdA==", Content: "c"}
	_, err := deserializeBlock(&xb)
	if err == nil {
		t.Error("Expected error for invalid UUID, got nil")
	}
}

func TestDeserializeBlock_InvalidBase64(t *testing.T) {
	t.Parallel()

	xb := xmlBlock{ID: testBlockID1.String(), Name: "T", State: "!!!invalid!!!", Content: "c"}
	_, err := deserializeBlock(&xb)
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}

func TestDeserializeBlock_EmptyState(t *testing.T) {
	t.Parallel()

	xb := xmlBlock{ID: testBlockID1.String(), Name: "T", State: "", Content: "c"}
	_, err := deserializeBlock(&xb)
	if !errors.Is(err, block.ErrEmptyBlockState) {
		t.Errorf("Expected ErrEmptyBlockState, got %v", err)
	}
}
