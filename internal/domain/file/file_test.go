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

func TestNewTypstFile(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	now := time.Now()
	b, _ := block.NewBlock(testBlockID, "Introduction", []byte("state"), "Hello")
	blocks := []block.Block{b}

	f, err := NewTypstFile(id, projectID, "document.typ", blocks, now)
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
	if len(f.Blocks()) != 1 || f.Blocks()[0].ID() != testBlockID {
		t.Errorf("Expected 1 block with ID 'block-1', got %v", f.Blocks())
	}
	if !f.UpdatedAt().Equal(now) {
		t.Errorf("Expected UpdatedAt %v, got %v", now, f.UpdatedAt())
	}

	// Test validation
	_, err = NewTypstFile(id, projectID, "", blocks, now)
	if !errors.Is(err, ErrEmptyFileName) {
		t.Errorf("Expected ErrEmptyFileName, got %v", err)
	}
}
