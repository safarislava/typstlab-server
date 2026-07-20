package serialization

import (
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

var testBlockID1 = uuid.MustParse("10000000-0000-0000-0000-000000000001")

const (
	testBlockName    = "Введение"
	testBlockContent = "= Введение\nТекст"
)

func TestSerializeBlock(t *testing.T) {
	t.Parallel()

	b, err := block.NewBlock(testBlockID1, testBlockName, testBlockContent)
	if err != nil {
		t.Fatalf("Failed to create block: %v", err)
	}

	xb := serializeBlock(b)

	if xb.ID != testBlockID1.String() {
		t.Errorf("Expected ID %s, got %s", testBlockID1, xb.ID)
	}
	if xb.Name != testBlockName {
		t.Errorf("Expected Name %q, got %q", testBlockName, xb.Name)
	}
	if xb.Content != testBlockContent {
		t.Errorf("Expected Content %q, got %q", testBlockContent, xb.Content)
	}
}

func TestDeserializeBlock(t *testing.T) {
	t.Parallel()

	xb := xmlBlock{
		ID:      testBlockID1.String(),
		Name:    testBlockName,
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
		t.Errorf("Expected Name %q, got %q", testBlockName, b.Name())
	}
	if b.Content() != testBlockContent {
		t.Errorf("Expected Content %q, got %q", testBlockContent, b.Content())
	}
}

func TestSerializeDeserializeBlock_Roundtrip(t *testing.T) {
	t.Parallel()

	original, err := block.NewBlock(testBlockID1, "Chapter", "= Chapter\nContent")
	if err != nil {
		t.Fatalf("Failed to create block: %v", err)
	}

	restored, err := deserializeBlock(new(serializeBlock(original)))
	if err != nil {
		t.Fatalf("deserializeBlock failed: %v", err)
	}

	if original.ID() != restored.ID() {
		t.Errorf("ID mismatch: %s vs %s", original.ID(), restored.ID())
	}
	if original.Name() != restored.Name() {
		t.Errorf("Name mismatch: %q vs %q", original.Name(), restored.Name())
	}
	if original.Content() != restored.Content() {
		t.Errorf("Content mismatch: %q vs %q", original.Content(), restored.Content())
	}
}

func TestDeserializeBlock_InvalidID(t *testing.T) {
	t.Parallel()

	xb := xmlBlock{ID: "not-a-uuid", Name: "T", Content: "c"}
	_, err := deserializeBlock(&xb)
	if err == nil {
		t.Error("Expected error for invalid UUID, got nil")
	}
}
