package block

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

var testBlockID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func TestNewBlock(t *testing.T) {
	t.Parallel()

	b, err := NewBlock(testBlockID, "Introduction", "Hello world")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if b.ID() != testBlockID {
		t.Errorf("Expected ID %s, got %s", testBlockID, b.ID())
	}
	if b.Name() != "Introduction" {
		t.Errorf("Expected Name 'Introduction', got %q", b.Name())
	}
	if b.Content() != "Hello world" {
		t.Errorf("Expected Content 'Hello world', got %q", b.Content())
	}

	// Test validation error
	_, err = NewBlock(uuid.Nil, "Introduction", "Hello")
	if !errors.Is(err, ErrEmptyBlockID) {
		t.Errorf("Expected ErrEmptyBlockID, got %v", err)
	}
}
