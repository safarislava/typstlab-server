package file

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/application/file/binary"
	"github.com/safarislava/typstlab-server/internal/application/file/typst"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

const (
	testNameSuccess = "success"
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

func setupTest(repo *mockRepository) (*Service, context.Context) {
	typstSvc := typst.NewService(repo)
	binarySvc := binary.NewService(repo)
	return NewService(repo, typstSvc, binarySvc), context.Background()
}

func TestService_RenameFile(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	typstID := uuid.New()
	binaryID := uuid.New()

	repo := newMockRepository()
	service, ctx := setupTest(repo)

	tf, _ := domainFile.NewTypstFile(typstID, projectID, "old.typ", nil, time.Now())
	bf, _ := domainFile.NewBinaryFile(binaryID, projectID, "old.png", []byte{1}, time.Now())

	_ = repo.SaveTypstFile(ctx, tf)
	_ = repo.SaveBinaryFile(ctx, bf)

	if err := service.RenameFile(ctx, typstID, "renamed.typ"); err != nil {
		t.Fatalf("unexpected error renaming typst: %v", err)
	}
	if tf.Name() != "renamed.typ" {
		t.Errorf("expected renamed typst name 'renamed.typ', got %s", tf.Name())
	}

	if err := service.RenameFile(ctx, binaryID, "renamed.png"); err != nil {
		t.Fatalf("unexpected error renaming binary: %v", err)
	}
	if bf.Name() != "renamed.png" {
		t.Errorf("expected renamed binary name 'renamed.png', got %s", bf.Name())
	}

	if err := service.RenameFile(ctx, uuid.New(), "random.typ"); err == nil {
		t.Error("expected error for non-existent file, got nil")
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newMockRepository()
			repo.deleteErr = tt.deleteErr
			service, ctx := setupTest(repo)

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
	service, ctx := setupTest(repo)

	tf, _ := domainFile.NewTypstFile(uuid.New(), projectID, "doc.typ", nil, time.Now())
	bf, _ := domainFile.NewBinaryFile(uuid.New(), projectID, "img.png", []byte{1, 2, 3}, time.Now())

	otherTF, _ := domainFile.NewTypstFile(uuid.New(), uuid.New(), "other.typ", nil, time.Now())

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

	repo.findErr = errors.New("find failed")
	_, err = service.ListFilesByProject(ctx, projectID)
	if err == nil {
		t.Error("expected find error, got nil")
	}
}
