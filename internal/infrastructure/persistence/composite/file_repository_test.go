package composite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/s3"
)

type mockPostgresStore struct {
	typstStore    map[uuid.UUID]*domainFile.TypstFile
	binaryStore   map[uuid.UUID]*domainFile.BinaryFile
	saveTypstErr  error
	findTypstErr  error
	saveBinaryErr error
	findBinaryErr error
	findProjErr   error
	deleteErr     error
	entriesErr    error
}

func newMockPostgresStore() *mockPostgresStore {
	return &mockPostgresStore{
		typstStore:  make(map[uuid.UUID]*domainFile.TypstFile),
		binaryStore: make(map[uuid.UUID]*domainFile.BinaryFile),
	}
}

func (m *mockPostgresStore) SaveTypstFile(_ context.Context, f *domainFile.TypstFile) error {
	if m.saveTypstErr != nil {
		return m.saveTypstErr
	}
	m.typstStore[f.ID()] = f
	return nil
}

func (m *mockPostgresStore) SaveBinaryFile(_ context.Context, f *domainFile.BinaryFile) error {
	if m.saveBinaryErr != nil {
		return m.saveBinaryErr
	}
	m.binaryStore[f.ID()] = f
	return nil
}

func (m *mockPostgresStore) FindTypstFileByID(_ context.Context, id uuid.UUID) (*domainFile.TypstFile, error) {
	if m.findTypstErr != nil {
		return nil, m.findTypstErr
	}
	f, ok := m.typstStore[id]
	if !ok {
		return nil, domainFile.ErrTypstFileNotFound
	}
	return f, nil
}

func (m *mockPostgresStore) FindBinaryFileByID(_ context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
	if m.findBinaryErr != nil {
		return nil, m.findBinaryErr
	}
	f, ok := m.binaryStore[id]
	if !ok {
		return nil, domainFile.ErrBinaryFileNotFound
	}
	return f, nil
}

func (m *mockPostgresStore) FindByProjectID(_ context.Context, _ uuid.UUID) ([]domainFile.File, error) {
	if m.findProjErr != nil {
		return nil, m.findProjErr
	}
	var files []domainFile.File
	for _, f := range m.typstStore {
		files = append(files, f)
	}
	for _, f := range m.binaryStore {
		files = append(files, f)
	}
	return files, nil
}

func (m *mockPostgresStore) DeleteFile(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.typstStore, id)
	delete(m.binaryStore, id)
	return nil
}

func (m *mockPostgresStore) FindEntriesByProjectID(_ context.Context, _ uuid.UUID) ([]*domainEntry.Entry, error) {
	if m.entriesErr != nil {
		return nil, m.entriesErr
	}
	var entries []*domainEntry.Entry
	for _, f := range m.typstStore {
		entry, _ := domainEntry.NewEntry(f.ID(), f.Name(), f.Type(), false, f.UpdatedAt())
		entries = append(entries, entry)
	}
	return entries, nil
}

type mockS3Storage struct {
	store       map[string][]byte
	uploadErr   error
	downloadErr error
	deleteErr   error
}

func newMockS3Storage() *mockS3Storage {
	return &mockS3Storage{
		store: make(map[string][]byte),
	}
}

func (m *mockS3Storage) Upload(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return errors.New("read error")
	}
	m.store[key] = data
	return nil
}

func (m *mockS3Storage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	data, ok := m.store[key]
	if !ok {
		return nil, s3.ErrObjectNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockS3Storage) Delete(_ context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.store, key)
	return nil
}

func (m *mockS3Storage) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://presigned.example.com", nil
}

func (m *mockS3Storage) EnsureBucket(_ context.Context) error {
	return nil
}

func TestNewFileRepository(t *testing.T) {
	t.Parallel()

	pg := newMockPostgresStore()
	storage := newMockS3Storage()

	if _, err := NewFileRepository(nil, storage); !errors.Is(err, ErrPostgresRepoRequired) {
		t.Errorf("expected ErrPostgresRepoRequired, got %v", err)
	}

	if _, err := NewFileRepository(pg, nil); !errors.Is(err, ErrS3StorageRequired) {
		t.Errorf("expected ErrS3StorageRequired, got %v", err)
	}

	repo, err := NewFileRepository(pg, storage)
	if err != nil || repo == nil {
		t.Fatalf("failed to create Composite FileRepository: %v", err)
	}
}

func TestFileRepository_TypstOperations(t *testing.T) {
	t.Parallel()

	pg := newMockPostgresStore()
	storage := newMockS3Storage()
	repo, _ := NewFileRepository(pg, storage)

	ctx := context.Background()
	tf, err := domainFile.NewTypstFile(uuid.New(), uuid.New(), "main.typ", []byte("state"), time.Now())
	if err != nil {
		t.Fatalf("failed to create typst file: %v", err)
	}

	// 1. Save Typst
	if saveErr := repo.SaveTypstFile(ctx, tf); saveErr != nil {
		t.Fatalf("unexpected save error: %v", saveErr)
	}

	// 2. Find Typst
	found, findErr := repo.FindTypstFileByID(ctx, tf.ID())
	if findErr != nil || found.ID() != tf.ID() {
		t.Fatalf("unexpected find error: %v", findErr)
	}

	// 3. Error paths
	pg.saveTypstErr = errors.New("save fail")
	if saveErr := repo.SaveTypstFile(ctx, tf); saveErr == nil {
		t.Error("expected save error, got nil")
	}

	pg.findTypstErr = errors.New("find fail")
	if _, findErr := repo.FindTypstFileByID(ctx, tf.ID()); findErr == nil {
		t.Error("expected find error, got nil")
	}
}

func TestFileRepository_BinaryOperations(t *testing.T) {
	t.Parallel()

	pg := newMockPostgresStore()
	storage := newMockS3Storage()
	repo, _ := NewFileRepository(pg, storage)

	ctx := context.Background()
	pID := uuid.New()
	fID := uuid.New()
	payload := []byte("image binary payload")

	bf, err := domainFile.NewBinaryFile(fID, pID, "photo.png", payload, time.Now())
	if err != nil {
		t.Fatalf("failed to create binary file: %v", err)
	}

	// 1. Save Binary (S3 upload + Postgres metadata)
	if saveErr := repo.SaveBinaryFile(ctx, bf); saveErr != nil {
		t.Fatalf("unexpected save error: %v", saveErr)
	}

	key := s3.BlobKey(pID, fID, "photo.png")
	if !bytes.Equal(storage.store[key], payload) {
		t.Errorf("expected S3 payload %s, got %s", payload, storage.store[key])
	}
	if pg.binaryStore[fID].Content() != nil {
		t.Errorf("expected nil content in Postgres metadata, got %v", pg.binaryStore[fID].Content())
	}

	// 2. Find Binary (Postgres metadata + S3 payload)
	found, findErr := repo.FindBinaryFileByID(ctx, fID)
	if findErr != nil {
		t.Fatalf("unexpected find error: %v", findErr)
	}
	if !bytes.Equal(found.Content(), payload) {
		t.Errorf("expected found content %s, got %s", payload, found.Content())
	}

	// 3. Find with S3 download error & fallback
	storage.downloadErr = errors.New("s3 download fail")
	if _, findErr := repo.FindBinaryFileByID(ctx, fID); findErr == nil {
		t.Error("expected download error when no fallback content exists, got nil")
	}

	// 4. Delete Binary
	storage.downloadErr = nil
	if delErr := repo.DeleteFile(ctx, fID); delErr != nil {
		t.Fatalf("unexpected delete error: %v", delErr)
	}
	if _, ok := storage.store[key]; ok {
		t.Error("expected S3 object to be deleted")
	}
}

func TestFileRepository_BinaryErrors(t *testing.T) {
	t.Parallel()

	pg := newMockPostgresStore()
	storage := newMockS3Storage()
	repo, _ := NewFileRepository(pg, storage)

	ctx := context.Background()
	bf, _ := domainFile.NewBinaryFile(uuid.New(), uuid.New(), "photo.png", []byte("data"), time.Now())

	// S3 upload error
	storage.uploadErr = errors.New("s3 upload error")
	if err := repo.SaveBinaryFile(ctx, bf); err == nil {
		t.Error("expected S3 upload error, got nil")
	}

	// Postgres save error
	storage.uploadErr = nil
	pg.saveBinaryErr = errors.New("postgres save error")
	if err := repo.SaveBinaryFile(ctx, bf); err == nil {
		t.Error("expected Postgres save error, got nil")
	}

	// Postgres find error
	pg.findBinaryErr = errors.New("postgres find error")
	if _, err := repo.FindBinaryFileByID(ctx, bf.ID()); err == nil {
		t.Error("expected Postgres find error, got nil")
	}

	// Postgres delete error
	pg.findBinaryErr = nil
	pg.deleteErr = errors.New("postgres delete error")
	if err := repo.DeleteFile(ctx, bf.ID()); err == nil {
		t.Error("expected Postgres delete error, got nil")
	}
}

func TestFileRepository_ProjectAndEntries(t *testing.T) {
	t.Parallel()

	pg := newMockPostgresStore()
	storage := newMockS3Storage()
	repo, _ := NewFileRepository(pg, storage)

	ctx := context.Background()
	pID := uuid.New()

	tf, _ := domainFile.NewTypstFile(uuid.New(), pID, "main.typ", []byte("state"), time.Now())
	_ = pg.SaveTypstFile(ctx, tf)

	files, err := repo.FindByProjectID(ctx, pID)
	if err != nil || len(files) != 1 {
		t.Fatalf("unexpected FindByProjectID result: files=%v, err=%v", files, err)
	}

	entries, err := repo.FindEntriesByProjectID(ctx, pID)
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected FindEntriesByProjectID result: entries=%v, err=%v", entries, err)
	}

	pg.findProjErr = errors.New("find proj fail")
	if _, err := repo.FindByProjectID(ctx, pID); err == nil {
		t.Error("expected FindByProjectID error, got nil")
	}

	pg.entriesErr = errors.New("entries fail")
	if _, err := repo.FindEntriesByProjectID(ctx, pID); err == nil {
		t.Error("expected FindEntriesByProjectID error, got nil")
	}
}
