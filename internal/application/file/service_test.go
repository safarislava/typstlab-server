package file

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

const (
	testFileNameTypst  = "test.typ"
	testFileNameBinary = "image.png"
	testNameSuccess    = "success"
	testNameSaveErr    = "save repository error"
)

type mockRepository struct {
	typstStore  map[uuid.UUID]*domainFile.TypstFile
	binaryStore map[uuid.UUID]*domainFile.BinaryFile
	saveErr     error
	findErr     error
	deleteErr   error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		typstStore:  make(map[uuid.UUID]*domainFile.TypstFile),
		binaryStore: make(map[uuid.UUID]*domainFile.BinaryFile),
	}
}

func (r *mockRepository) SaveTypstFile(_ context.Context, f *domainFile.TypstFile) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.typstStore[f.ID()] = f
	return nil
}

func (r *mockRepository) SaveBinaryFile(_ context.Context, f *domainFile.BinaryFile) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.binaryStore[f.ID()] = f
	return nil
}

func (r *mockRepository) FindTypstFileByID(_ context.Context, id uuid.UUID) (*domainFile.TypstFile, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	f, ok := r.typstStore[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return f, nil
}

func (r *mockRepository) FindBinaryFileByID(_ context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	f, ok := r.binaryStore[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return f, nil
}

func (r *mockRepository) FindByProjectID(_ context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	var res []domainFile.File
	for _, f := range r.typstStore {
		if f.ProjectID() == projectID {
			res = append(res, f)
		}
	}
	for _, f := range r.binaryStore {
		if f.ProjectID() == projectID {
			res = append(res, f)
		}
	}
	return res, nil
}

func (r *mockRepository) DeleteFile(_ context.Context, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.typstStore, id)
	delete(r.binaryStore, id)
	return nil
}

type mockMerger struct {
	mergedState  []byte
	mergedBlocks []block.Block
	mergeErr     error
}

func (m *mockMerger) MergeFile(_, _ []byte) ([]byte, []block.Block, error) {
	if m.mergeErr != nil {
		return nil, nil, m.mergeErr
	}
	return m.mergedState, m.mergedBlocks, nil
}

func setupTest(repo *mockRepository, merger *mockMerger) (UseCase, context.Context) {
	return NewService(repo, merger), context.Background()
}

func TestService_UploadTypstFile_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadTypstFileRequest{
		ID:        fileID,
		ProjectID: projectID,
		Name:      testFileNameTypst,
	}
	f, err := service.UploadTypstFile(ctx, req)
	if err != nil {
		t.Fatalf("UploadTypstFile() unexpected error: %v", err)
	}
	if f.ID() != fileID || f.Name() != testFileNameTypst || f.ProjectID() != projectID {
		t.Errorf("incorrect file fields: %+v", f)
	}
	if len(f.State()) != 0 {
		t.Errorf("expected empty state, got %s", f.State())
	}
}

func TestService_UploadTypstFile_SuccessWithContent(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	b, _ := block.NewBlock(uuid.New(), "Intro", "Content")
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadTypstFileRequest{
		ID:        fileID,
		ProjectID: projectID,
		Name:      testFileNameTypst,
		State:     []byte("initial-crdt-state"),
		Blocks:    []block.Block{b},
	}
	f, err := service.UploadTypstFile(ctx, req)
	if err != nil {
		t.Fatalf("UploadTypstFile() unexpected error: %v", err)
	}
	if f.ID() != fileID || f.Name() != testFileNameTypst || f.ProjectID() != projectID {
		t.Errorf("incorrect file fields: %+v", f)
	}
	if string(f.State()) != "initial-crdt-state" {
		t.Errorf("expected state 'initial-crdt-state', got %s", f.State())
	}
	if len(f.Blocks()) != 1 {
		t.Errorf("expected 1 block, got %d", len(f.Blocks()))
	}
}

func TestService_UploadTypstFile_ValidationError(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadTypstFileRequest{
		ID:        uuid.Nil,
		ProjectID: projectID,
		Name:      testFileNameTypst,
	}
	_, err := service.UploadTypstFile(ctx, req)
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestService_UploadTypstFile_SaveErr(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	repo := newMockRepository()
	repo.saveErr = errors.New("save failed")
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadTypstFileRequest{
		ID:        fileID,
		ProjectID: projectID,
		Name:      testFileNameTypst,
	}
	_, err := service.UploadTypstFile(ctx, req)
	if err == nil {
		t.Error("expected save error, got nil")
	}
}

func TestService_UploadBinaryFile_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadBinaryFileRequest{
		ID:        fileID,
		ProjectID: projectID,
		Name:      testFileNameBinary,
		Content:   []byte{1, 2},
	}
	f, err := service.UploadBinaryFile(ctx, req)
	if err != nil {
		t.Fatalf("UploadBinaryFile() unexpected error: %v", err)
	}
	if f.ID() != fileID || f.Name() != testFileNameBinary || f.ProjectID() != projectID {
		t.Errorf("incorrect file: %+v", f)
	}
	if !bytes.Equal(f.Content(), []byte{1, 2}) {
		t.Errorf("expected content %v, got %v", []byte{1, 2}, f.Content())
	}
}

func TestService_UploadBinaryFile_ValidationError(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadBinaryFileRequest{
		ID:        uuid.Nil,
		ProjectID: projectID,
		Name:      testFileNameBinary,
	}
	_, err := service.UploadBinaryFile(ctx, req)
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestService_UploadBinaryFile_SaveErr(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	repo := newMockRepository()
	repo.saveErr = errors.New("save failed")
	service, ctx := setupTest(repo, &mockMerger{})

	req := &UploadBinaryFileRequest{
		ID:        fileID,
		ProjectID: projectID,
		Name:      testFileNameBinary,
	}
	_, err := service.UploadBinaryFile(ctx, req)
	if err == nil {
		t.Error("expected save error, got nil")
	}
}

func TestService_GetTypstFile(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	fileID := uuid.New()
	tf, err := domainFile.NewTypstFile(fileID, uuid.New(), "doc.typ", []byte("state"), nil, time.Now())
	if err != nil {
		t.Fatalf("failed to create typst file: %v", err)
	}
	if saveErr := repo.SaveTypstFile(ctx, tf); saveErr != nil {
		t.Fatalf("failed to save typst file: %v", saveErr)
	}

	f, err := service.GetTypstFile(ctx, fileID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID() != fileID {
		t.Errorf("expected file id %v, got %v", fileID, f.ID())
	}

	_, err = service.GetTypstFile(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestService_GetBinaryFile(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	fileID := uuid.New()
	bf, err := domainFile.NewBinaryFile(fileID, uuid.New(), "img.png", []byte{1}, time.Now())
	if err != nil {
		t.Fatalf("failed to create binary file: %v", err)
	}
	if saveErr := repo.SaveBinaryFile(ctx, bf); saveErr != nil {
		t.Fatalf("failed to save binary file: %v", saveErr)
	}

	f, err := service.GetBinaryFile(ctx, fileID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID() != fileID {
		t.Errorf("expected file id %v, got %v", fileID, f.ID())
	}

	_, err = service.GetBinaryFile(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestService_ApplyFileChanges(t *testing.T) {
	t.Parallel()

	fileID := uuid.New()
	req := ApplyFileChangesRequest{
		FileID: fileID,
		Delta:  []byte("delta"),
	}

	b, err := block.NewBlock(uuid.New(), "Intro", "Content")
	if err != nil {
		t.Fatalf("failed to create block: %v", err)
	}

	tests := []struct {
		name      string
		findErr   error
		mergeErr  error
		saveErr   error
		wantErr   bool
		checkFunc func(t *testing.T, f *domainFile.TypstFile)
	}{
		{
			name:    testNameSuccess,
			wantErr: false,
			checkFunc: func(t *testing.T, f *domainFile.TypstFile) {
				if string(f.State()) != "updated" {
					t.Errorf("expected state 'updated', got %s", f.State())
				}
			},
		},
		{
			name:    "find error",
			findErr: errors.New("find failed"),
			wantErr: true,
		},
		{
			name:     "merge error",
			mergeErr: errors.New("merge failed"),
			wantErr:  true,
		},
		{
			name:    testNameSaveErr,
			saveErr: errors.New("save failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runApplyFileChangesSubtest(t, fileID, req, b, tt.findErr, tt.mergeErr, tt.saveErr, tt.wantErr, tt.checkFunc)
		})
	}
}

func runApplyFileChangesSubtest(
	t *testing.T,
	fileID uuid.UUID,
	req ApplyFileChangesRequest,
	b block.Block,
	findErr, mergeErr, saveErr error,
	wantErr bool,
	checkFunc func(t *testing.T, f *domainFile.TypstFile),
) {
	t.Helper()
	repo := newMockRepository()
	repo.findErr = findErr

	merger := &mockMerger{
		mergedState:  []byte("updated"),
		mergedBlocks: []block.Block{b},
		mergeErr:     mergeErr,
	}

	service, ctx := setupTest(repo, merger)

	tf, err := domainFile.NewTypstFile(fileID, uuid.New(), "doc.typ", []byte("initial"), nil, time.Now())
	if err != nil {
		t.Fatalf("failed to create typst file: %v", err)
	}
	if saveErrHelper := repo.SaveTypstFile(ctx, tf); saveErrHelper != nil {
		t.Fatalf("failed to save typst file: %v", saveErrHelper)
	}

	repo.saveErr = saveErr

	f, err := service.ApplyFileChanges(ctx, req)
	if (err != nil) != wantErr {
		t.Fatalf("ApplyFileChanges() error = %v, wantErr = %v", err, wantErr)
	}
	if err == nil && checkFunc != nil {
		checkFunc(t, f)
	}
}

func TestService_DeleteFile(t *testing.T) {
	t.Parallel()
	fileID := uuid.New()

	tests := []struct {
		name      string
		deleteErr error
		wantErr   bool
	}{
		{
			name:    testNameSuccess,
			wantErr: false,
		},
		{
			name:      "delete error",
			deleteErr: errors.New("delete failed"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newMockRepository()
			repo.deleteErr = tt.deleteErr
			service, ctx := setupTest(repo, &mockMerger{})

			err := service.DeleteFile(ctx, fileID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeleteFile() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_ListFilesByProject(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()

	repo := newMockRepository()
	service, ctx := setupTest(repo, &mockMerger{})

	tf, _ := domainFile.NewTypstFile(uuid.New(), projectID, "doc.typ", nil, nil, time.Now())
	bf, _ := domainFile.NewBinaryFile(uuid.New(), projectID, "img.png", []byte{1, 2, 3}, time.Now())

	otherTF, _ := domainFile.NewTypstFile(uuid.New(), uuid.New(), "other.typ", nil, nil, time.Now())

	_ = repo.SaveTypstFile(ctx, tf)
	_ = repo.SaveBinaryFile(ctx, bf)
	_ = repo.SaveTypstFile(ctx, otherTF)

	files, err := service.ListFilesByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("ListFilesByProject() error = %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}
