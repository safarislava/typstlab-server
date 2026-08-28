package file

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type mockFileManager struct {
	file       *domainFile.TypstFile
	getErr     error
	saveErr    error
	listErr    error
	renameErr  error
	deleteErr  error
	files      []domainFile.File
	renamedIDs []uuid.UUID
	deletedIDs []uuid.UUID
	savedFile  *domainFile.TypstFile
}

func (m *mockFileManager) GetTypstFile(_ context.Context, _ uuid.UUID) (*domainFile.TypstFile, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.file, nil
}

func (m *mockFileManager) SaveTypstFile(_ context.Context, f *domainFile.TypstFile) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedFile = f
	return nil
}

func (m *mockFileManager) ListFilesByProject(_ context.Context, _ uuid.UUID) ([]domainFile.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.files, nil
}

func (m *mockFileManager) RenameFile(_ context.Context, fileID uuid.UUID, _ string) error {
	if m.renameErr != nil {
		return m.renameErr
	}
	m.renamedIDs = append(m.renamedIDs, fileID)
	return nil
}

func (m *mockFileManager) DeleteFile(_ context.Context, fileID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedIDs = append(m.deletedIDs, fileID)
	return nil
}

type mockFileMerger struct {
	newState []byte
	blocks   []block.Block
	mergeErr error
}

func (m *mockFileMerger) MergeFile(_, _ []byte) ([]byte, []block.Block, error) {
	if m.mergeErr != nil {
		return nil, nil, m.mergeErr
	}
	return m.newState, m.blocks, nil
}

type mockDeltaCalculator struct {
	computeDeltaFunc func(serverState, clientStateVector []byte) ([]byte, error)
}

func (m *mockDeltaCalculator) ComputeDelta(serverState, clientStateVector []byte) ([]byte, error) {
	if m.computeDeltaFunc != nil {
		return m.computeDeltaFunc(serverState, clientStateVector)
	}
	return []byte("mock-delta"), nil
}

func TestService_ApplyFileChanges_Success(t *testing.T) {
	t.Parallel()
	fileID := uuid.New()
	projectID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, projectID, "doc.typ", []byte("old-state"), nil, time.Now())

	b, _ := block.NewBlock(uuid.New(), "intro", "hello")
	merger := &mockFileMerger{
		newState: []byte("new-state"),
		blocks:   []block.Block{b},
	}
	fileMgr := &mockFileManager{file: tf}
	svc := NewService(fileMgr, merger, &mockDeltaCalculator{})

	updated, err := svc.ApplyFileChanges(context.Background(), ApplyFileChangesRequest{
		FileID: fileID,
		Delta:  []byte("delta"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(updated.State()) != "new-state" {
		t.Errorf("expected state 'new-state', got %s", updated.State())
	}
	if fileMgr.savedFile != tf {
		t.Errorf("expected saved file to match tf")
	}
}

func TestService_ApplyMetadataMutations(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	renameID := uuid.New()
	deleteID := uuid.New()

	renameFile, _ := domainFile.NewTypstFile(renameID, projectID, "old.typ", nil, nil, time.Now())
	deleteFile, _ := domainFile.NewTypstFile(deleteID, projectID, "delete.typ", nil, nil, time.Now())
	fileMgr := &mockFileManager{files: []domainFile.File{renameFile, deleteFile}}

	renamedEntry, _ := domainEntry.NewEntry(renameID, "new.typ", domainFile.TypeTypst, false, time.Now())
	deletedEntry, _ := domainEntry.NewEntry(deleteID, "delete.typ", domainFile.TypeTypst, true, time.Now())
	meta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{renamedEntry, deletedEntry})

	svc := NewService(fileMgr, &mockFileMerger{}, &mockDeltaCalculator{})

	err := svc.ApplyMetadataMutations(context.Background(), []domainFile.File{renameFile, deleteFile}, meta)
	if err != nil {
		t.Fatalf("unexpected mutation error: %v", err)
	}

	if len(fileMgr.renamedIDs) != 1 || fileMgr.renamedIDs[0] != renameID {
		t.Errorf("expected file %s to be renamed, got %+v", renameID, fileMgr.renamedIDs)
	}
	if len(fileMgr.deletedIDs) != 1 || fileMgr.deletedIDs[0] != deleteID {
		t.Errorf("expected file %s to be deleted, got %+v", deleteID, fileMgr.deletedIDs)
	}
}

func TestService_GenerateContentInstructions(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	offlineFileID := uuid.New()

	serverFile, _ := domainFile.NewTypstFile(fileID, projectID, "doc.typ", []byte("server-state"), nil, time.Now())
	serverFiles := []domainFile.File{serverFile}

	serverEntry, _ := domainEntry.NewEntry(fileID, "doc.typ", domainFile.TypeTypst, false, time.Now())
	offlineEntry, _ := domainEntry.NewEntry(offlineFileID, "offline.typ", domainFile.TypeTypst, false, time.Now())
	meta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{serverEntry, offlineEntry})

	deltaCalc := &mockDeltaCalculator{
		computeDeltaFunc: func(_, _ []byte) ([]byte, error) {
			return []byte("delta-bytes"), nil
		},
	}

	svc := NewService(&mockFileManager{}, &mockFileMerger{}, deltaCalc)

	instructions, err := svc.GenerateContentInstructions(serverFiles, meta, map[uuid.UUID][]byte{
		fileID: []byte("client-vector"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instructions))
	}
	if instructions[0].Action != ActionUpload || instructions[0].FileID != offlineFileID {
		t.Errorf("unexpected upload instruction: %+v", instructions[0])
	}
	if instructions[1].Action != ActionApplyChanges || instructions[1].FileID != fileID {
		t.Errorf("unexpected apply_changes instruction: %+v", instructions[1])
	}
}

func TestService_ListFilesByProject_Error(t *testing.T) {
	t.Parallel()
	fileMgr := &mockFileManager{listErr: errors.New("failed to list")}
	svc := NewService(fileMgr, &mockFileMerger{}, &mockDeltaCalculator{})

	_, err := svc.ListFilesByProject(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestService_ApplyFileChanges_Error(t *testing.T) {
	t.Parallel()
	fileMgr := &mockFileManager{getErr: errors.New("failed to get")}
	svc := NewService(fileMgr, &mockFileMerger{}, &mockDeltaCalculator{})

	_, err := svc.ApplyFileChanges(context.Background(), ApplyFileChangesRequest{
		FileID: uuid.New(),
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
}
