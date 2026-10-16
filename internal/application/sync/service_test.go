package sync

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

type mockMetaRepo struct {
	entries []*domainEntry.Entry
	err     error
}

func (m *mockMetaRepo) FindEntriesByProjectID(_ context.Context, _ uuid.UUID) ([]*domainEntry.Entry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}

type mockFileService struct {
	files      []domainFile.File
	listErr    error
	deleteErr  error
	saveErr    error
	deletedIDs []uuid.UUID
	savedFiles []domainFile.File
}

func (m *mockFileService) ListFilesByProject(_ context.Context, _ uuid.UUID) ([]domainFile.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.files, nil
}

func (m *mockFileService) DeleteFile(_ context.Context, fileID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedIDs = append(m.deletedIDs, fileID)
	return nil
}

func (m *mockFileService) Save(_ context.Context, f domainFile.File) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedFiles = append(m.savedFiles, f)
	return nil
}

type mockTypstFileService struct {
	file      *domainFile.TypstFile
	getErr    error
	saveErr   error
	savedFile *domainFile.TypstFile
}

func (m *mockTypstFileService) GetByID(_ context.Context, _ uuid.UUID) (*domainFile.TypstFile, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.file, nil
}

func (m *mockTypstFileService) Save(_ context.Context, f *domainFile.TypstFile) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedFile = f
	return nil
}

type mockCRDTEngine struct {
	mergedState       []byte
	mergeErr          error
	computedDelta     []byte
	computeErr        error
	syncMetaDelta     []byte
	syncMetaResult    *domainMeta.Metadata
	syncMetaErr       error
	computeDeltaCalls int
}

func (m *mockCRDTEngine) MergeFile(_, _ []byte) ([]byte, error) {
	if m.mergeErr != nil {
		return nil, m.mergeErr
	}
	return m.mergedState, nil
}

func (m *mockCRDTEngine) ComputeDelta(_, _ []byte) ([]byte, error) {
	m.computeDeltaCalls++
	if m.computeErr != nil {
		return nil, m.computeErr
	}
	return m.computedDelta, nil
}

func (m *mockCRDTEngine) SyncMetadata(
	_ uuid.UUID,
	_ *domainMeta.Metadata,
	_, _ []byte,
) ([]byte, *domainMeta.Metadata, error) {
	if m.syncMetaErr != nil {
		return nil, nil, m.syncMetaErr
	}
	return m.syncMetaDelta, m.syncMetaResult, nil
}

func TestService_SyncInit_Success(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	typstID := uuid.New()
	uploadID := uuid.New()
	deletedID := uuid.New()

	typstFile, _ := domainFile.NewTypstFile(typstID, projectID, "doc.typ", []byte("server-content"), time.Now())
	oldFile, _ := domainFile.NewTypstFile(deletedID, projectID, "old.typ", nil, time.Now())

	metaEntry1, _ := domainEntry.NewEntry(typstID, "renamed.typ", domainFile.TypeTypst, false, time.Now())
	metaEntry2, _ := domainEntry.NewEntry(uploadID, "new.typ", domainFile.TypeTypst, false, time.Now())
	metaEntry3, _ := domainEntry.NewEntry(deletedID, "old.typ", domainFile.TypeTypst, true, time.Now())

	updatedMeta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{metaEntry1, metaEntry2, metaEntry3})

	metaRepo := &mockMetaRepo{entries: []*domainEntry.Entry{}}
	fileService := &mockFileService{
		files: []domainFile.File{typstFile, oldFile},
	}
	typstService := &mockTypstFileService{}
	crdtEngine := &mockCRDTEngine{
		syncMetaDelta:  []byte("meta-delta"),
		syncMetaResult: updatedMeta,
		computedDelta:  []byte("typst-delta"),
	}

	svc := NewService(metaRepo, fileService, typstService, crdtEngine)

	req := SyncInitRequest{
		ProjectID:           projectID,
		MetadataStateVector: []byte("client-sv"),
		ContentVectors: map[uuid.UUID][]byte{
			typstID: []byte("client-vector"),
		},
	}

	resp, err := svc.SyncInit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(resp.MetadataDelta) != "meta-delta" {
		t.Errorf("expected meta-delta, got %s", string(resp.MetadataDelta))
	}

	verifySyncInitSavedFiles(t, fileService, typstID, deletedID)
	verifySyncInitInstructions(t, resp.Instructions, uploadID, typstID)
}

func verifySyncInitSavedFiles(t *testing.T, fs *mockFileService, typstID, deletedID uuid.UUID) {
	t.Helper()
	if len(fs.savedFiles) != 1 {
		t.Fatalf("expected 1 saved file, got %d", len(fs.savedFiles))
	}
	f := fs.savedFiles[0]
	if f.ID() != typstID || f.Type() != domainFile.TypeTypst || f.Name() != "renamed.typ" {
		t.Errorf("unexpected saved file: ID=%s, type=%s, name=%s", f.ID(), f.Type(), f.Name())
	}
	if len(fs.deletedIDs) != 1 || fs.deletedIDs[0] != deletedID {
		t.Errorf("expected deletedID %s, got %v", deletedID, fs.deletedIDs)
	}
}

func verifySyncInitInstructions(t *testing.T, instructions []Instruction, uploadID, typstID uuid.UUID) {
	t.Helper()
	if len(instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instructions))
	}
	instMap := make(map[uuid.UUID]Instruction, len(instructions))
	for _, inst := range instructions {
		instMap[inst.FileID] = inst
	}

	uploadInst, hasUpload := instMap[uploadID]
	if !hasUpload || uploadInst.Action != ActionUpload {
		t.Errorf("expected upload instruction for %s, got %+v", uploadID, uploadInst)
	}

	applyInst, hasApply := instMap[typstID]
	if !hasApply || applyInst.Action != ActionApplyChanges || string(applyInst.Delta) != "typst-delta" {
		t.Errorf("expected apply_changes instruction for %s, got %+v", typstID, applyInst)
	}
}

func TestService_ApplyMetadataUpdate_PartialSync(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	fileID := uuid.New()

	serverFile, _ := domainFile.NewTypstFile(fileID, projectID, "old-name.typ", []byte("content"), time.Now())
	entry, _ := domainEntry.NewEntry(fileID, "new-name.typ", domainFile.TypeTypst, false, time.Now())
	updatedMeta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{entry})

	metaRepo := &mockMetaRepo{}
	fileService := &mockFileService{files: []domainFile.File{serverFile}}
	typstService := &mockTypstFileService{}
	crdtEngine := &mockCRDTEngine{
		syncMetaResult: updatedMeta,
	}

	svc := NewService(metaRepo, fileService, typstService, crdtEngine)

	err := svc.ApplyMetadataUpdate(context.Background(), projectID, []byte("incremental-delta"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mutation was applied
	if len(fileService.savedFiles) != 1 {
		t.Fatalf("expected 1 saved file, got %d", len(fileService.savedFiles))
	}
	if fileService.savedFiles[0].Name() != "new-name.typ" {
		t.Errorf("expected rename to new-name.typ, got %s", fileService.savedFiles[0].Name())
	}

	// Verify that ComputeDelta was NEVER called (no full instruction scan)
	if crdtEngine.computeDeltaCalls != 0 {
		t.Errorf("expected 0 calls to ComputeDelta during partial metadata sync, got %d", crdtEngine.computeDeltaCalls)
	}

	// Empty delta validation
	if err := svc.ApplyMetadataUpdate(context.Background(), projectID, nil); err == nil {
		t.Error("expected error on empty delta")
	}
}

func TestService_ApplyCRDTUpdate_PartialSync(t *testing.T) {
	t.Parallel()

	fileID := uuid.New()
	projectID := uuid.New()
	typstFile, _ := domainFile.NewTypstFile(fileID, projectID, "doc.typ", []byte("old-state"), time.Now())

	metaRepo := &mockMetaRepo{}
	fileService := &mockFileService{}
	typstService := &mockTypstFileService{file: typstFile}
	crdtEngine := &mockCRDTEngine{mergedState: []byte("merged-state")}

	svc := NewService(metaRepo, fileService, typstService, crdtEngine)

	err := svc.ApplyCRDTUpdate(context.Background(), fileID, []byte("file-delta"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if typstService.savedFile == nil {
		t.Fatal("expected file to be saved")
	}
	if string(typstService.savedFile.State()) != "merged-state" {
		t.Errorf("expected merged-state, got %s", string(typstService.savedFile.State()))
	}

	// Validation errors
	if err := svc.ApplyCRDTUpdate(context.Background(), uuid.Nil, []byte("delta")); err == nil {
		t.Error("expected error for nil fileID")
	}
	if err := svc.ApplyCRDTUpdate(context.Background(), fileID, nil); err == nil {
		t.Error("expected error for empty delta")
	}
}

func TestService_Errors(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	fileID := uuid.New()

	t.Run("load metadata error", func(t *testing.T) {
		t.Parallel()
		svc := NewService(
			&mockMetaRepo{err: errors.New("db error")},
			&mockFileService{},
			&mockTypstFileService{},
			&mockCRDTEngine{},
		)
		if _, err := svc.SyncInit(context.Background(), SyncInitRequest{ProjectID: projectID}); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("sync metadata crdt error", func(t *testing.T) {
		t.Parallel()
		svc := NewService(
			&mockMetaRepo{},
			&mockFileService{},
			&mockTypstFileService{},
			&mockCRDTEngine{syncMetaErr: errors.New("crdt error")},
		)
		if _, err := svc.SyncInit(context.Background(), SyncInitRequest{ProjectID: projectID}); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("list server files error", func(t *testing.T) {
		t.Parallel()
		svc := NewService(
			&mockMetaRepo{},
			&mockFileService{listErr: errors.New("list error")},
			&mockTypstFileService{},
			&mockCRDTEngine{},
		)
		if _, err := svc.SyncInit(context.Background(), SyncInitRequest{ProjectID: projectID}); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("typst file not found error", func(t *testing.T) {
		t.Parallel()
		svc := NewService(
			&mockMetaRepo{},
			&mockFileService{},
			&mockTypstFileService{getErr: errors.New("not found")},
			&mockCRDTEngine{},
		)
		if err := svc.ApplyCRDTUpdate(context.Background(), fileID, []byte("delta")); err == nil {
			t.Error("expected error")
		}
	})
}
