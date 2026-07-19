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

func TestTypstFile_FindBlockByID(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	b1, _ := block.NewBlock(testBlockID, "Introduction", []byte("state1"), "Hello")
	f, _ := NewTypstFile(id, projectID, "doc.typ", []block.Block{b1}, time.Now())

	found, err := f.FindBlockByID(testBlockID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID() != testBlockID {
		t.Errorf("expected block ID %v, got %v", testBlockID, found.ID())
	}

	_, err = f.FindBlockByID(uuid.New())
	if !errors.Is(err, ErrBlockNotFound) {
		t.Errorf("expected ErrBlockNotFound, got %v", err)
	}
}

func TestTypstFile_UpdateBlock(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	b1, _ := block.NewBlock(testBlockID, "Introduction", []byte("state1"), "Hello")
	f, _ := NewTypstFile(id, projectID, "doc.typ", []block.Block{b1}, time.Now())

	err := f.UpdateBlock(testBlockID, []byte("new-state"), "Updated Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := f.FindBlockByID(testBlockID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Content() != "Updated Hello" {
		t.Errorf("expected 'Updated Hello', got %q", updated.Content())
	}
	if !bytes.Equal(updated.State(), []byte("new-state")) {
		t.Errorf("expected 'new-state', got %s", updated.State())
	}

	// Update non-existent block
	err = f.UpdateBlock(uuid.New(), []byte("state"), "content")
	if !errors.Is(err, ErrBlockNotFound) {
		t.Errorf("expected ErrBlockNotFound, got %v", err)
	}
}
