package typst

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

const (
	testFileName = "test.typ"
	docFileName  = "doc.typ"
)

type mockRepository struct {
	store   map[uuid.UUID]*domainFile.TypstFile
	saveErr error
	findErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		store: make(map[uuid.UUID]*domainFile.TypstFile),
	}
}

func (r *mockRepository) SaveTypstFile(_ context.Context, f *domainFile.TypstFile) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.store[f.ID()] = f
	return nil
}

func (r *mockRepository) FindTypstFileByID(_ context.Context, id uuid.UUID) (*domainFile.TypstFile, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	f, ok := r.store[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return f, nil
}

func TestService_Upload_Success(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	fileID := uuid.New()
	repo := newMockRepository()
	svc := NewService(repo)

	req := &UploadRequest{
		ID:        fileID,
		ProjectID: projectID,
		Name:      testFileName,
	}
	f, err := svc.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID() != fileID || f.Name() != testFileName || f.ProjectID() != projectID {
		t.Errorf("incorrect file fields: %+v", f)
	}
}

func TestService_Upload_ValidationError(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	svc := NewService(repo)

	req := &UploadRequest{
		ID:        uuid.Nil,
		ProjectID: uuid.New(),
		Name:      testFileName,
	}
	_, err := svc.Upload(context.Background(), req)
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestService_Upload_SaveErr(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	repo.saveErr = errors.New("save failed")
	svc := NewService(repo)

	req := &UploadRequest{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Name:      testFileName,
	}
	_, err := svc.Upload(context.Background(), req)
	if err == nil {
		t.Error("expected save error, got nil")
	}
}

func TestService_Save(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	svc := NewService(repo)

	tf, _ := domainFile.NewTypstFile(uuid.New(), uuid.New(), docFileName, nil, nil, time.Now())
	if err := svc.Save(context.Background(), tf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo.saveErr = errors.New("save failed")
	if err := svc.Save(context.Background(), tf); err == nil {
		t.Error("expected save error, got nil")
	}
}

func TestService_GetByID(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	svc := NewService(repo)

	fileID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, uuid.New(), docFileName, []byte("state"), nil, time.Now())
	_ = repo.SaveTypstFile(context.Background(), tf)

	f, err := svc.GetByID(context.Background(), fileID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID() != fileID {
		t.Errorf("expected file id %v, got %v", fileID, f.ID())
	}

	_, err = svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
