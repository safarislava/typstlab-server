package file

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

var testBlockID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func TestNewBinaryFile(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	content := []byte("binary-data")
	now := time.Now()

	f, err := NewBinaryFile(id, projectID, "image.png", content, now)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if f.ID() != id {
		t.Errorf("Expected ID %s, got %s", id, f.ID())
	}
	if f.ProjectID() != projectID {
		t.Errorf("Expected ProjectID %s, got %s", projectID, f.ProjectID())
	}
	if f.Name() != "image.png" {
		t.Errorf("Expected Name 'image.png', got %q", f.Name())
	}
	if f.Type() != TypeBinary {
		t.Errorf("Expected Type 'binary', got %q", f.Type())
	}
	if !bytes.Equal(f.Content(), content) {
		t.Errorf("Expected Content %s, got %s", content, f.Content())
	}
	if !f.UpdatedAt().Equal(now) {
		t.Errorf("Expected UpdatedAt %v, got %v", now, f.UpdatedAt())
	}

	// Test validation
	_, err = NewBinaryFile(uuid.Nil, projectID, "image.png", content, now)
	if !errors.Is(err, ErrEmptyFileID) {
		t.Errorf("Expected ErrEmptyFileID, got %v", err)
	}
}

func TestNewTypstFile_Success(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	now := time.Now()
	b, _ := block.NewBlock(testBlockID, "Introduction", "Hello")
	blocks := []block.Block{b}
	state := []byte("global-state")

	f, err := NewTypstFile(id, projectID, "document.typ", state, blocks, now)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if f.ID() != id {
		t.Errorf("Expected ID %s, got %s", id, f.ID())
	}
	if f.ProjectID() != projectID {
		t.Errorf("Expected ProjectID %s, got %s", projectID, f.ProjectID())
	}
	if f.Name() != "document.typ" {
		t.Errorf("Expected Name 'document.typ', got %q", f.Name())
	}
	if f.Type() != TypeTypst {
		t.Errorf("Expected Type 'typst', got %q", f.Type())
	}
	if !bytes.Equal(f.State(), state) {
		t.Errorf("Expected State %s, got %s", state, f.State())
	}
	if len(f.Blocks()) != 1 || f.Blocks()[0].ID() != testBlockID {
		t.Errorf("Expected 1 block with ID 'block-1', got %v", f.Blocks())
	}
	if !f.UpdatedAt().Equal(now) {
		t.Errorf("Expected UpdatedAt %v, got %v", now, f.UpdatedAt())
	}
}

func TestNewTypstFile_ValidationError(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	now := time.Now()
	b, _ := block.NewBlock(testBlockID, "Introduction", "Hello")
	blocks := []block.Block{b}
	state := []byte("global-state")

	_, err := NewTypstFile(id, projectID, "", state, blocks, now)
	if !errors.Is(err, ErrEmptyFileName) {
		t.Errorf("Expected ErrEmptyFileName, got %v", err)
	}
}

func TestTypstFile_UpdateState(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	b1, _ := block.NewBlock(testBlockID, "Introduction", "Hello")
	f, _ := NewTypstFile(id, projectID, "doc.typ", []byte("initial"), []block.Block{b1}, time.Now())

	b2, _ := block.NewBlock(uuid.New(), "New Section", "Some content")
	newState := []byte("updated-state")
	err := f.UpdateState(newState, []block.Block{b1, b2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(f.State(), newState) {
		t.Errorf("expected state %s, got %s", newState, f.State())
	}
	if len(f.Blocks()) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(f.Blocks()))
	}

	err = f.UpdateState(nil, nil)
	if err == nil {
		t.Error("expected error for nil state, got nil")
	}
}
