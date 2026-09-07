package binary

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

const (
	testImageName = "image.png"
)

type mockRepository struct {
	store   map[uuid.UUID]*domainFile.BinaryFile
	saveErr error
	findErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		store: make(map[uuid.UUID]*domainFile.BinaryFile),
	}
}

func (r *mockRepository) SaveBinaryFile(_ context.Context, f *domainFile.BinaryFile) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.store[f.ID()] = f
	return nil
}

func (r *mockRepository) FindBinaryFileByID(_ context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
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
		Name:      testImageName,
		Content:   []byte{1, 2, 3},
	}
	f, err := svc.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID() != fileID || f.Name() != testImageName || f.ProjectID() != projectID {
		t.Errorf("incorrect file fields: %+v", f)
	}
	if !bytes.Equal(f.Content(), []byte{1, 2, 3}) {
		t.Errorf("expected content [1 2 3], got %v", f.Content())
	}
}

func TestService_Upload_ValidationError(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	svc := NewService(repo)

	req := &UploadRequest{
		ID:        uuid.Nil,
		ProjectID: uuid.New(),
		Name:      testImageName,
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
		Name:      testImageName,
		Content:   []byte{1},
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

	bf, _ := domainFile.NewBinaryFile(uuid.New(), uuid.New(), testImageName, []byte{1}, time.Now())
	if err := svc.Save(context.Background(), bf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo.saveErr = errors.New("save failed")
	if err := svc.Save(context.Background(), bf); err == nil {
		t.Error("expected save error, got nil")
	}
}

func TestService_GetByID(t *testing.T) {
	t.Parallel()
	repo := newMockRepository()
	svc := NewService(repo)

	fileID := uuid.New()
	bf, _ := domainFile.NewBinaryFile(fileID, uuid.New(), testImageName, []byte{1}, time.Now())
	_ = repo.SaveBinaryFile(context.Background(), bf)

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
