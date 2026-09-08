package file

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

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
	state := []byte("global-state")

	f, err := NewTypstFile(id, projectID, "document.typ", state, now)
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
	if !f.UpdatedAt().Equal(now) {
		t.Errorf("Expected UpdatedAt %v, got %v", now, f.UpdatedAt())
	}
}

func TestNewTypstFile_ValidationError(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	now := time.Now()
	state := []byte("global-state")

	_, err := NewTypstFile(id, projectID, "", state, now)
	if !errors.Is(err, ErrEmptyFileName) {
		t.Errorf("Expected ErrEmptyFileName, got %v", err)
	}
}

func TestTypstFile_UpdateState(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	projectID := uuid.New()
	f, _ := NewTypstFile(id, projectID, "doc.typ", []byte("initial"), time.Now())

	newState := []byte("updated-state")
	err := f.UpdateState(newState)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(f.State(), newState) {
		t.Errorf("expected state %s, got %s", newState, f.State())
	}

	err = f.UpdateState(nil)
	if err == nil {
		t.Error("expected error for nil state, got nil")
	}
}

func TestBinaryFile_Rename(t *testing.T) {
	t.Parallel()

	f, _ := NewBinaryFile(uuid.New(), uuid.New(), "old.png", []byte("content"), time.Now())
	if err := f.Rename("new.png"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name() != "new.png" {
		t.Errorf("expected name 'new.png', got %s", f.Name())
	}

	if err := f.Rename(""); !errors.Is(err, ErrEmptyFileName) {
		t.Errorf("expected ErrEmptyFileName, got %v", err)
	}
}

func TestTypstFile_Rename(t *testing.T) {
	t.Parallel()

	f, _ := NewTypstFile(uuid.New(), uuid.New(), "old.typ", []byte("state"), time.Now())
	if err := f.Rename("new.typ"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name() != "new.typ" {
		t.Errorf("expected name 'new.typ', got %s", f.Name())
	}

	if err := f.Rename(""); !errors.Is(err, ErrEmptyFileName) {
		t.Errorf("expected ErrEmptyFileName, got %v", err)
	}
}
