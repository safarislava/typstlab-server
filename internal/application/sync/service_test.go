package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	syncFile "github.com/safarislava/typstlab-server/internal/application/sync/file"
	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type mockMetadataSyncer struct {
	delta   []byte
	meta    *domainMeta.Metadata
	syncErr error
}

func (m *mockMetadataSyncer) SyncMetadata(
	_ context.Context,
	_ uuid.UUID,
	_, _ []byte,
) ([]byte, *domainMeta.Metadata, error) {
	if m.syncErr != nil {
		return nil, nil, m.syncErr
	}
	return m.delta, m.meta, nil
}

type mockFileSyncer struct {
	files        []domainFile.File
	listErr      error
	applyMetaErr error
	instErr      error
	instructions []syncFile.Instruction
	appliedMeta  *domainMeta.Metadata
	changedFile  *domainFile.TypstFile
	changeErr    error
}

func (m *mockFileSyncer) ListFilesByProject(_ context.Context, _ uuid.UUID) ([]domainFile.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.files, nil
}

func (m *mockFileSyncer) ApplyFileChanges(_ context.Context, _ syncFile.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	if m.changeErr != nil {
		return nil, m.changeErr
	}
	return m.changedFile, nil
}

func (m *mockFileSyncer) ApplyMetadataMutations(_ context.Context, _ []domainFile.File, meta *domainMeta.Metadata) error {
	m.appliedMeta = meta
	return m.applyMetaErr
}

func (m *mockFileSyncer) GenerateContentInstructions(
	_ []domainFile.File,
	_ *domainMeta.Metadata,
	_ map[uuid.UUID][]byte,
) ([]syncFile.Instruction, error) {
	if m.instErr != nil {
		return nil, m.instErr
	}
	return m.instructions, nil
}

func TestSyncService_Sync_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()

	entry, _ := domainEntry.NewEntry(fileID, "doc.typ", domainFile.TypeTypst, false, time.Now())
	meta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{entry})

	metaSyncer := &mockMetadataSyncer{
		delta: []byte("server-metadata-delta"),
		meta:  meta,
	}

	fileSyncer := &mockFileSyncer{
		instructions: []syncFile.Instruction{
			{Action: ActionApplyChanges, FileID: fileID, Delta: []byte("delta")},
		},
	}

	svc := NewService(metaSyncer, fileSyncer)

	resp, err := svc.Sync(context.Background(), projectID, &Request{
		MetadataDelta: []byte("client-delta"),
	})
	if err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	if string(resp.MetadataDelta) != "server-metadata-delta" {
		t.Errorf("expected delta 'server-metadata-delta', got %s", resp.MetadataDelta)
	}
	if len(resp.Instructions) != 1 {
		t.Errorf("expected 1 instruction, got %d", len(resp.Instructions))
	}
	if fileSyncer.appliedMeta != meta {
		t.Errorf("expected metadata to be applied")
	}
}

func TestSyncService_Sync_MetadataError(t *testing.T) {
	t.Parallel()
	metaSyncer := &mockMetadataSyncer{syncErr: errors.New("meta sync failed")}
	svc := NewService(metaSyncer, &mockFileSyncer{})

	_, err := svc.Sync(context.Background(), uuid.New(), &Request{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSyncService_Sync_ListFilesError(t *testing.T) {
	t.Parallel()
	metaSyncer := &mockMetadataSyncer{}
	fileSyncer := &mockFileSyncer{listErr: errors.New("list failed")}
	svc := NewService(metaSyncer, fileSyncer)

	_, err := svc.Sync(context.Background(), uuid.New(), &Request{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSyncService_ApplyFileChanges(t *testing.T) {
	t.Parallel()
	fileID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, uuid.New(), "doc.typ", nil, nil, time.Now())
	fileSyncer := &mockFileSyncer{changedFile: tf}
	svc := NewService(&mockMetadataSyncer{}, fileSyncer)

	res, err := svc.ApplyFileChanges(context.Background(), ApplyFileChangesRequest{
		FileID: fileID,
		Delta:  []byte("delta"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != tf {
		t.Errorf("expected changed file")
	}
}
