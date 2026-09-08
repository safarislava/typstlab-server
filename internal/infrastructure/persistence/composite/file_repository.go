package composite

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/s3"
)

var (
	ErrPostgresRepoRequired = errors.New("postgres file repository is required")
	ErrS3StorageRequired    = errors.New("s3 storage is required")
)

// PostgresStore defines the contract required from PostgreSQL file persistence.
type PostgresStore interface {
	SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error
	SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error
	FindTypstFileByID(ctx context.Context, id uuid.UUID) (*domainFile.TypstFile, error)
	FindBinaryFileByID(ctx context.Context, id uuid.UUID) (*domainFile.BinaryFile, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error
	FindEntriesByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domainEntry.Entry, error)
}

// FileRepository orchestrates file metadata in PostgreSQL and binary payloads in S3.
type FileRepository struct {
	postgres PostgresStore
	s3       s3.Storage
}

func NewFileRepository(postgres PostgresStore, s3Storage s3.Storage) (*FileRepository, error) {
	if postgres == nil {
		return nil, ErrPostgresRepoRequired
	}
	if s3Storage == nil {
		return nil, ErrS3StorageRequired
	}
	return &FileRepository{
		postgres: postgres,
		s3:       s3Storage,
	}, nil
}

// SaveTypstFile delegates Typst file persistence directly to PostgreSQL.
func (r *FileRepository) SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error {
	if err := r.postgres.SaveTypstFile(ctx, f); err != nil {
		return fmt.Errorf("failed to save typst file: %w", err)
	}
	return nil
}

// FindTypstFileByID retrieves a Typst file directly from PostgreSQL.
func (r *FileRepository) FindTypstFileByID(ctx context.Context, id uuid.UUID) (*domainFile.TypstFile, error) {
	tf, err := r.postgres.FindTypstFileByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}
	return tf, nil
}

// SaveBinaryFile uploads the binary payload to S3 and writes metadata to PostgreSQL.
func (r *FileRepository) SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error {
	content := f.Content()
	contentType := http.DetectContentType(content)
	key := s3.BlobKey(f.ProjectID(), f.ID(), f.Name())

	if err := r.s3.Upload(ctx, key, bytes.NewReader(content), int64(len(content)), contentType); err != nil {
		return fmt.Errorf("failed to upload blob to s3: %w", err)
	}

	metaFile, err := domainFile.NewBinaryFile(f.ID(), f.ProjectID(), f.Name(), nil, f.UpdatedAt())
	if err != nil {
		return fmt.Errorf("failed to create binary metadata: %w", err)
	}

	if err := r.postgres.SaveBinaryFile(ctx, metaFile); err != nil {
		return fmt.Errorf("failed to save binary metadata to postgres: %w", err)
	}

	return nil
}

// FindBinaryFileByID fetches file metadata from PostgreSQL and downloads payload from S3.
func (r *FileRepository) FindBinaryFileByID(ctx context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
	metaFile, err := r.postgres.FindBinaryFileByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find binary metadata: %w", err)
	}

	key := s3.BlobKey(metaFile.ProjectID(), metaFile.ID(), metaFile.Name())
	rc, err := r.s3.Download(ctx, key)
	if err != nil {
		if len(metaFile.Content()) > 0 {
			return metaFile, nil
		}
		return nil, fmt.Errorf("failed to download blob from s3: %w", err)
	}
	defer func() { _ = rc.Close() }()

	content, readErr := io.ReadAll(rc)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read blob from s3: %w", readErr)
	}

	f, err := domainFile.NewBinaryFile(metaFile.ID(), metaFile.ProjectID(), metaFile.Name(), content, metaFile.UpdatedAt())
	if err != nil {
		return nil, fmt.Errorf("failed to construct binary file: %w", err)
	}

	return f, nil
}

// FindByProjectID retrieves file listings from PostgreSQL.
func (r *FileRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	files, err := r.postgres.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find files by project: %w", err)
	}
	return files, nil
}

// DeleteFile deletes binary payload from S3 (if binary) and record from PostgreSQL.
func (r *FileRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	if bf, err := r.postgres.FindBinaryFileByID(ctx, id); err == nil {
		key := s3.BlobKey(bf.ProjectID(), bf.ID(), bf.Name())
		_ = r.s3.Delete(ctx, key)
	}

	if err := r.postgres.DeleteFile(ctx, id); err != nil {
		return fmt.Errorf("failed to delete file from postgres: %w", err)
	}

	return nil
}

// FindEntriesByProjectID retrieves metadata entries for all files from PostgreSQL.
func (r *FileRepository) FindEntriesByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domainEntry.Entry, error) {
	entries, err := r.postgres.FindEntriesByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find entries by project: %w", err)
	}
	return entries, nil
}
