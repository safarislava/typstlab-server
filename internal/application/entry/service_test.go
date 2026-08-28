package entry

import (
	"testing"
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

func TestEntryService(t *testing.T) {
	t.Parallel()

	svc := NewService()
	id := uuid.New()
	now := time.Now()

	e, err := svc.CreateEntry(id, "document.typ", domainFile.TypeTypst, false, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if e.ID() != id || e.Name() != "document.typ" {
		t.Errorf("unexpected entry fields: %+v", e)
	}

	err = svc.Rename(e, "renamed.typ")
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if e.Name() != "renamed.typ" {
		t.Errorf("expected name 'renamed.typ', got %s", e.Name())
	}

	svc.Delete(e)
	if !e.IsDeleted() {
		t.Error("expected entry to be marked deleted")
	}
}
