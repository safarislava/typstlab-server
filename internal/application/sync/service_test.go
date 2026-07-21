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

func (m *mockFileRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.files, nil
}

func (m *mockFileRepository) IsDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.deletedFiles == nil {
		return false, nil
	}
	return m.deletedFiles[id], nil
}

func TestSyncService_Sync_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()

	// 1. Prepare Yjs document states for apply_changes test
	serverDoc := crdt.New()
	serverDoc.Transact(func(txn *crdt.Transaction) {
		text := txn.GetText("block:test")
		text.Insert(txn, 0, "Hello World", nil)
	})
	serverState := serverDoc.EncodeStateAsUpdate()

	clientDoc := crdt.New() // empty client doc
	clientStateVector := crdt.EncodeStateVectorV1(clientDoc)

	// 2. Prepare server files
	fileID1 := uuid.New() // Match client file, no changes
	fileID2 := uuid.New() // Match client file, different name -> rename
	fileID3 := uuid.New() // Match client file, missing updates -> apply_changes
	fileID4 := uuid.New() // Present on server only -> download
	fileID5 := uuid.New() // Conflict name target
	fileID8 := uuid.New() // Deleted on server -> delete

	file1, _ := domainFile.NewTypstFile(fileID1, projectID, "file1.typ", nil, nil, time.Now())
	file2, _ := domainFile.NewTypstFile(fileID2, projectID, "file2-server.typ", nil, nil, time.Now())
	file3, _ := domainFile.NewTypstFile(fileID3, projectID, "file3.typ", serverState, nil, time.Now())
	file4, _ := domainFile.NewTypstFile(fileID4, projectID, "file4.typ", nil, nil, time.Now())
	file5, _ := domainFile.NewTypstFile(fileID5, projectID, "file5-conflict.typ", nil, nil, time.Now())

	repo := &mockFileRepository{
		files:        []domainFile.File{file1, file2, file3, file4, file5},
		deletedFiles: map[uuid.UUID]bool{fileID8: true},
	}
	service := NewService(repo)

	// 3. Client manifest request
	clientFile1 := FileRequest{
		ID:   fileID1,
		Name: "file1.typ",
		Type: domainFile.TypeTypst,
	}
	clientFile2 := FileRequest{ // Name mismatch
		ID:   fileID2,
		Name: "file2-client.typ",
		Type: domainFile.TypeTypst,
	}
	clientFile3 := FileRequest{ // State vector sent
		ID:    fileID3,
		Name:  "file3.typ",
		Type:  domainFile.TypeTypst,
		State: clientStateVector,
	}
	clientFile6 := FileRequest{ // Offline created file
		ID:   uuid.New(),
		Name: "file6-offline.typ",
		Type: domainFile.TypeTypst,
	}
	clientFile7 := FileRequest{ // Offline created file with name conflict
		ID:   uuid.New(),
		Name: "file5-conflict.typ",
		Type: domainFile.TypeTypst,
	}
	clientFile8 := FileRequest{ // Deleted on server
		ID:   fileID8,
		Name: "file8-deleted.typ",
		Type: domainFile.TypeTypst,
	}

	request := &Request{
		Files: []FileRequest{clientFile1, clientFile2, clientFile3, clientFile6, clientFile7, clientFile8},
	}

	response, err := service.Sync(context.Background(), projectID, request)
	if err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	// 4. Verify instructions
	instructions := response.Instructions

	var foundRenameF2, foundApplyChangesF3, foundUploadF6, foundRenameF7, foundUploadF7, foundDownloadF4, foundDeleteF8 bool

	for _, instruction := range instructions {
		switch instruction.FileID {
		case fileID2:
			if instruction.Action == ActionRename && instruction.NewName == "file2-server.typ" {
				foundRenameF2 = true
			}
		case fileID3:
			if instruction.Action == ActionApplyChanges && len(instruction.Delta) > 0 {
				foundApplyChangesF3 = true
			}
		case clientFile6.ID:
			if instruction.Action == ActionUpload {
				foundUploadF6 = true
			}
		case clientFile7.ID:
			if instruction.Action == ActionRename && instruction.NewName == "file5-conflict_conflict.typ" {
				foundRenameF7 = true
			}
			if instruction.Action == ActionUpload {
				foundUploadF7 = true
			}
		case fileID4:
			if instruction.Action == ActionDownload {
				foundDownloadF4 = true
			}
		case fileID8:
			if instruction.Action == ActionDelete {
				foundDeleteF8 = true
			}
		}
	}

	if !foundRenameF2 {
		t.Error("Expected rename instruction for f2")
	}
	if !foundApplyChangesF3 {
		t.Error("Expected apply_changes instruction for f3")
	}
	if !foundUploadF6 {
		t.Error("Expected upload instruction for f6")
	}
	if !foundRenameF7 {
		t.Error("Expected rename instruction for f7 conflict")
	}
	if !foundUploadF7 {
		t.Error("Expected upload instruction for f7")
	}
	if !foundDownloadF4 {
		t.Error("Expected download instruction for f4")
	}
	if !foundDeleteF8 {
		t.Error("Expected delete instruction for f8")
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
