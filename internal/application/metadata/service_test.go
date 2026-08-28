package metadata

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type mockRepository struct {
	entries []*domainEntry.Entry
	findErr error
}

func (m *mockRepository) FindEntriesByProjectID(_ context.Context, _ uuid.UUID) ([]*domainEntry.Entry, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.entries, nil
}

func TestService_GetMetadata_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	entry, _ := domainEntry.NewEntry(uuid.New(), "doc.typ", domainFile.TypeTypst, false, time.Now())

	repo := &mockRepository{entries: []*domainEntry.Entry{entry}}
	svc := NewService(repo)

	meta, err := svc.GetMetadata(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.ProjectID() != projectID {
		t.Errorf("expected project ID %s, got %s", projectID, meta.ProjectID())
	}
	if len(meta.Entries()) != 1 {
		t.Errorf("expected 1 entry, got %d", len(meta.Entries()))
	}
}

func TestService_GetMetadata_Error(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{findErr: errors.New("query failed")}
	svc := NewService(repo)

	_, err := svc.GetMetadata(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestService_CreateMetadata(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	entry, _ := domainEntry.NewEntry(uuid.New(), "doc.typ", domainFile.TypeTypst, false, time.Now())

	svc := NewService(&mockRepository{})
	meta, err := svc.CreateMetadata(projectID, []*domainEntry.Entry{entry})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.ProjectID() != projectID {
		t.Errorf("expected project ID %s, got %s", projectID, meta.ProjectID())
	}
}
