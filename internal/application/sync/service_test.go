package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type mockFileRepository struct {
	fileApp.Repository
	files        []domainFile.File
	deletedFiles map[uuid.UUID]bool
	findErr      error
}

func (m *mockFileRepository) FindByProjectID(context.Context, uuid.UUID) ([]domainFile.File, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.files, nil
}

func (m *mockFileRepository) IsDeleted(_ context.Context, id uuid.UUID) (bool, error) {
	if m.deletedFiles == nil {
		return false, nil
	}
	return m.deletedFiles[id], nil
}

func TestSyncService_Sync_RenameOnMismatch(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()

	file, _ := domainFile.NewTypstFile(fileID, projectID, "file2-server.typ", nil, nil, time.Now())
	repo := &mockFileRepository{files: []domainFile.File{file}}
	service := NewService(repo)

	req := &Request{
		Files: []FileRequest{
			{ID: fileID, Name: "file2-client.typ", Type: domainFile.TypeTypst},
		},
	}
	resp, err := service.Sync(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instructions) != 1 || resp.Instructions[0].Action != ActionRename || resp.Instructions[0].NewName != "file2-server.typ" {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestSyncService_Sync_ApplyChanges(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()

	serverDoc := crdt.New()
	serverDoc.Transact(func(txn *crdt.Transaction) {
		text := txn.GetText("block:test")
		text.Insert(txn, 0, "Hello World", nil)
	})
	serverState := serverDoc.EncodeStateAsUpdate()

	clientDoc := crdt.New()
	clientStateVector := crdt.EncodeStateVectorV1(clientDoc)

	file, _ := domainFile.NewTypstFile(fileID, projectID, "file.typ", serverState, nil, time.Now())
	repo := &mockFileRepository{files: []domainFile.File{file}}
	service := NewService(repo)

	req := &Request{
		Files: []FileRequest{
			{ID: fileID, Name: "file.typ", Type: domainFile.TypeTypst, State: clientStateVector},
		},
	}
	resp, err := service.Sync(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instructions) != 1 || resp.Instructions[0].Action != ActionApplyChanges || len(resp.Instructions[0].Delta) == 0 {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestSyncService_Sync_UploadOfflineCreated(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	offlineID := uuid.New()

	repo := &mockFileRepository{}
	service := NewService(repo)

	req := &Request{
		Files: []FileRequest{
			{ID: offlineID, Name: "new.typ", Type: domainFile.TypeTypst},
		},
	}
	resp, err := service.Sync(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instructions) != 1 || resp.Instructions[0].Action != ActionUpload || resp.Instructions[0].FileID != offlineID {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestSyncService_Sync_UploadOfflineConflict(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	offlineID := uuid.New()
	existingID := uuid.New()

	existingFile, _ := domainFile.NewTypstFile(existingID, projectID, "conflict.typ", nil, nil, time.Now())
	repo := &mockFileRepository{files: []domainFile.File{existingFile}}
	service := NewService(repo)

	req := &Request{
		Files: []FileRequest{
			{ID: offlineID, Name: "conflict.typ", Type: domainFile.TypeTypst},
		},
	}
	resp, err := service.Sync(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instructions) != 3 {
		t.Fatalf("expected 3 instructions, got: %+v", resp.Instructions)
	}
}

func TestSyncService_Sync_Download(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	serverFileID := uuid.New()

	serverFile, _ := domainFile.NewTypstFile(serverFileID, projectID, "server.typ", nil, nil, time.Now())
	repo := &mockFileRepository{files: []domainFile.File{serverFile}}
	service := NewService(repo)

	req := &Request{Files: []FileRequest{}}
	resp, err := service.Sync(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instructions) != 1 || resp.Instructions[0].Action != ActionDownload || resp.Instructions[0].FileID != serverFileID {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestSyncService_Sync_Delete(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	deletedFileID := uuid.New()

	repo := &mockFileRepository{
		deletedFiles: map[uuid.UUID]bool{deletedFileID: true},
	}
	service := NewService(repo)

	req := &Request{
		Files: []FileRequest{
			{ID: deletedFileID, Name: "deleted.typ", Type: domainFile.TypeTypst},
		},
	}
	resp, err := service.Sync(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instructions) != 1 || resp.Instructions[0].Action != ActionDelete || resp.Instructions[0].FileID != deletedFileID {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestSyncService_Sync_RepositoryError(t *testing.T) {
	t.Parallel()
	repo := &mockFileRepository{
		findErr: errors.New("database failure"),
	}
	service := NewService(repo)

	_, err := service.Sync(context.Background(), uuid.New(), &Request{})
	if err == nil {
		t.Error("Expected repository query error, got nil")
	}
}
