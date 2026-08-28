package file

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type UploadTypstFileRequest struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	State     []byte
	Blocks    []block.Block
}

type UploadBinaryFileRequest struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	Content   []byte
}

type Response struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	UpdatedAt time.Time
}

type Repository interface {
	SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error
	SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error
	FindTypstFileByID(ctx context.Context, id uuid.UUID) (*domainFile.TypstFile, error)
	FindBinaryFileByID(ctx context.Context, id uuid.UUID) (*domainFile.BinaryFile, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error
	IsDeleted(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) UploadTypstFile(ctx context.Context, req *UploadTypstFileRequest) (*domainFile.TypstFile, error) {
	f, err := domainFile.NewTypstFile(req.ID, req.ProjectID, req.Name, req.State, req.Blocks, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to upload typst file: %w", err)
	}

	if err := s.repo.SaveTypstFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save typst file: %w", err)
	}

	return f, nil
}

func (s *Service) UploadBinaryFile(ctx context.Context, req *UploadBinaryFileRequest) (*domainFile.BinaryFile, error) {
	f, err := domainFile.NewBinaryFile(req.ID, req.ProjectID, req.Name, req.Content, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to upload binary file: %w", err)
	}

	if err := s.repo.SaveBinaryFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save binary file: %w", err)
	}

	return f, nil
}

func (s *Service) SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error {
	if err := s.repo.SaveTypstFile(ctx, f); err != nil {
		return fmt.Errorf("failed to save typst file: %w", err)
	}
	return nil
}

func (s *Service) SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error {
	if err := s.repo.SaveBinaryFile(ctx, f); err != nil {
		return fmt.Errorf("failed to save binary file: %w", err)
	}
	return nil
}

func (s *Service) GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error) {
	f, err := s.repo.FindTypstFileByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}

	return f, nil
}

func (s *Service) GetBinaryFile(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error) {
	f, err := s.repo.FindBinaryFileByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find binary file: %w", err)
	}

	return f, nil
}

func (s *Service) RenameFile(ctx context.Context, fileID uuid.UUID, newName string) error {
	tf, errTypst := s.repo.FindTypstFileByID(ctx, fileID)
	if errTypst == nil {
		if err := tf.Rename(newName); err != nil {
			return fmt.Errorf("failed to rename typst file %s: %w", fileID, err)
		}
		return s.SaveTypstFile(ctx, tf)
	}

	bf, errBinary := s.repo.FindBinaryFileByID(ctx, fileID)
	if errBinary == nil {
		if err := bf.Rename(newName); err != nil {
			return fmt.Errorf("failed to rename binary file %s: %w", fileID, err)
		}
		return s.SaveBinaryFile(ctx, bf)
	}

	return fmt.Errorf("file not found: %s", fileID)
}

func (s *Service) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	if err := s.repo.DeleteFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (s *Service) ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	files, err := s.repo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find files by project: %w", err)
	}
	return files, nil
}
