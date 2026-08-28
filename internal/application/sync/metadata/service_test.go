package metadata

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type mockMetadataManager struct {
	meta   *domainMeta.Metadata
	getErr error
}

func (m *mockMetadataManager) GetMetadata(_ context.Context, _ uuid.UUID) (*domainMeta.Metadata, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.meta, nil
}

type mockMetadataSyncer struct {
	delta   []byte
	meta    *domainMeta.Metadata
	syncErr error
}

func (m *mockMetadataSyncer) SyncMetadata(
	_ uuid.UUID,
	_ *domainMeta.Metadata,
	_, _ []byte,
) ([]byte, *domainMeta.Metadata, error) {
	if m.syncErr != nil {
		return nil, nil, m.syncErr
	}
	return m.delta, m.meta, nil
}

func TestService_SyncMetadata_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()

	entry, _ := domainEntry.NewEntry(fileID, "doc.typ", domainFile.TypeTypst, false, time.Now())
	meta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{entry})

	metaMgr := &mockMetadataManager{meta: meta}
	syncer := &mockMetadataSyncer{
		delta: []byte("delta-bytes"),
		meta:  meta,
	}

	svc := NewService(metaMgr, syncer)

	delta, resMeta, err := svc.SyncMetadata(context.Background(), projectID, []byte("client-delta"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(delta) != "delta-bytes" {
		t.Errorf("expected delta 'delta-bytes', got %s", delta)
	}
	if resMeta != meta {
		t.Errorf("expected meta %+v, got %+v", meta, resMeta)
	}
}

func TestService_SyncMetadata_GetMetadataError(t *testing.T) {
	t.Parallel()
	metaMgr := &mockMetadataManager{getErr: errors.New("meta fetch error")}
	svc := NewService(metaMgr, &mockMetadataSyncer{})

	_, _, err := svc.SyncMetadata(context.Background(), uuid.New(), nil, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestService_SyncMetadata_SyncerError(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	meta, _ := domainMeta.NewMetadata(projectID, nil)
	metaMgr := &mockMetadataManager{meta: meta}
	syncer := &mockMetadataSyncer{syncErr: errors.New("sync error")}

	svc := NewService(metaMgr, syncer)

	_, _, err := svc.SyncMetadata(context.Background(), projectID, nil, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
