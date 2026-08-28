package metadata

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	f1, _ := domainEntry.NewEntry(uuid.New(), "file1.typ", domainFile.TypeTypst, false, time.Now())
	f2, _ := domainEntry.NewEntry(uuid.New(), "file2.typ", domainFile.TypeTypst, true, time.Now())

	meta, err := NewMetadata(projectID, []*domainEntry.Entry{f1, f2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.ProjectID() != projectID {
		t.Errorf("expected project ID %s, got %s", projectID, meta.ProjectID())
	}

	if len(meta.Entries()) != 2 {
		t.Errorf("expected 2 entries, got %d", len(meta.Entries()))
	}

	active := meta.ActiveEntries()
	if len(active) != 1 || active[0].ID() != f1.ID() {
		t.Errorf("expected 1 active entry with ID %s, got %+v", f1.ID(), active)
	}

	gotF1, exists := meta.Get(f1.ID())
	if !exists || gotF1.Name() != "file1.typ" {
		t.Errorf("expected to find file1.typ, got %+v", gotF1)
	}

	f3, _ := domainEntry.NewEntry(uuid.New(), "file3.png", domainFile.TypeBinary, false, time.Now())
	meta.Set(f3)
	if len(meta.Entries()) != 3 {
		t.Errorf("expected 3 entries after Set, got %d", len(meta.Entries()))
	}

	_, err = NewMetadata(uuid.Nil, nil)
	if !errors.Is(err, ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID, got %v", err)
	}
}
