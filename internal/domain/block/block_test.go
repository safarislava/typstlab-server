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

	crdt := []byte("crdt-bytes")
	b, err := NewBlock(testBlockID, "Introduction", crdt, "Hello world")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if b.ID() != testBlockID {
		t.Errorf("Expected ID %s, got %s", testBlockID, b.ID())
	}
	if b.Name() != "Introduction" {
		t.Errorf("Expected Name 'Introduction', got %q", b.Name())
	}
	if !bytes.Equal(b.CRDTState(), crdt) {
		t.Errorf("Expected CRDTState %s, got %s", crdt, b.CRDTState())
	}
	if b.Content() != "Hello world" {
		t.Errorf("Expected Content 'Hello world', got %q", b.Content())
	}

	// Test validation error
	_, err = NewBlock(uuid.Nil, "Introduction", crdt, "Hello")
	if !errors.Is(err, ErrEmptyBlockID) {
		t.Errorf("Expected ErrEmptyBlockID, got %v", err)
	}

	// Test validation error for Nil CRDT
	_, err = NewBlock(testBlockID, "Introduction", nil, "Hello")
	if !errors.Is(err, ErrEmptyBlockCrdt) {
		t.Errorf("Expected ErrEmptyBlockCrdt, got %v", err)
	}
}

func TestNewBlock_CRDTState_DefensiveCopy(t *testing.T) {
	t.Parallel()

	crdt := []byte("original")
	b, err := NewBlock(testBlockID, "Introduction", crdt, "Hello")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	crdt[0] = 'X'

	if bytes.Equal(b.CRDTState(), crdt) {
		t.Error("Modifying the input crdtState slice mutated the block's internal state")
	}
}

func TestBlock_CRDTState_Immutability(t *testing.T) {
	t.Parallel()

	crdt := []byte("original")
	b, err := NewBlock(testBlockID, "Introduction", crdt, "Hello")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	state := b.CRDTState()
	state[0] = 'X'

	if bytes.Equal(b.CRDTState(), state) {
		t.Error("Modifying the returned CRDTState slice mutated the block's internal state")
	}
}
