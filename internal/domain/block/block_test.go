package block

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/uuid"
)

var testBlockID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func TestNewBlock(t *testing.T) {
	t.Parallel()

	state := []byte("state-bytes")
	b, err := NewBlock(testBlockID, "Introduction", state, "Hello world")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if b.ID() != testBlockID {
		t.Errorf("Expected ID %s, got %s", testBlockID, b.ID())
	}
	if b.Name() != "Introduction" {
		t.Errorf("Expected Name 'Introduction', got %q", b.Name())
	}
	if !bytes.Equal(b.State(), state) {
		t.Errorf("Expected State %s, got %s", state, b.State())
	}
	if b.Content() != "Hello world" {
		t.Errorf("Expected Content 'Hello world', got %q", b.Content())
	}

	// Test validation error
	_, err = NewBlock(uuid.Nil, "Introduction", state, "Hello")
	if !errors.Is(err, ErrEmptyBlockID) {
		t.Errorf("Expected ErrEmptyBlockID, got %v", err)
	}

	// Test validation error for nil state
	_, err = NewBlock(testBlockID, "Introduction", nil, "Hello")
	if !errors.Is(err, ErrEmptyBlockState) {
		t.Errorf("Expected ErrEmptyBlockState, got %v", err)
	}
}

func TestNewBlock_State_DefensiveCopy(t *testing.T) {
	t.Parallel()

	state := []byte("original")
	b, err := NewBlock(testBlockID, "Introduction", state, "Hello")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	state[0] = 'X'

	if bytes.Equal(b.State(), state) {
		t.Error("Modifying the input state slice mutated the block's internal state")
	}
}

func TestBlock_State_Immutability(t *testing.T) {
	t.Parallel()

	state := []byte("original")
	b, err := NewBlock(testBlockID, "Introduction", state, "Hello")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	returned := b.State()
	returned[0] = 'X'

	if bytes.Equal(b.State(), returned) {
		t.Error("Modifying the returned State slice mutated the block's internal state")
	}
}
