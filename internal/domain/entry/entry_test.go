package entry

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

func TestNewEntry_Success(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	now := time.Now()

	e, err := NewEntry(id, "document.typ", domainFile.TypeTypst, false, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if e.ID() != id {
		t.Errorf("expected ID %s, got %s", id, e.ID())
	}
	if e.Name() != "document.typ" {
		t.Errorf("expected Name %q, got %q", "document.typ", e.Name())
	}
	if e.Type() != domainFile.TypeTypst {
		t.Errorf("expected Type %s, got %s", domainFile.TypeTypst, e.Type())
	}
	if e.IsDeleted() {
		t.Error("expected IsDeleted to be false")
	}
	if !e.UpdatedAt().Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, e.UpdatedAt())
	}
}

func TestNewEntry_ValidationErrors(t *testing.T) {
	t.Parallel()

	_, err := NewEntry(uuid.Nil, "document.typ", domainFile.TypeTypst, false, time.Now())
	if !errors.Is(err, ErrEmptyID) {
		t.Errorf("expected ErrEmptyID, got %v", err)
	}

	_, err = NewEntry(uuid.New(), "", domainFile.TypeTypst, false, time.Now())
	if !errors.Is(err, ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestEntry_RenameAndMarkDeleted(t *testing.T) {
	t.Parallel()

	e, _ := NewEntry(uuid.New(), "old.typ", domainFile.TypeTypst, false, time.Now())

	err := e.Rename("new.typ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Name() != "new.typ" {
		t.Errorf("expected name 'new.typ', got %s", e.Name())
	}

	err = e.Rename("")
	if !errors.Is(err, ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}

	e.MarkDeleted()
	if !e.IsDeleted() {
		t.Error("expected IsDeleted to be true")
	}
}
