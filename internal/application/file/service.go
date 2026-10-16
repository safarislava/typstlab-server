package file

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type TypstService interface {
	Save(ctx context.Context, f *domainFile.TypstFile) error
	GetByID(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
}

type BinaryService interface {
	Save(ctx context.Context, f *domainFile.BinaryFile) error
	GetByID(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error)
}

type Repository interface {
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	repo   Repository
	typst  TypstService
	binary BinaryService
}

func NewService(repo Repository, typstService TypstService, binaryService BinaryService) *Service {
	return &Service{
		repo:   repo,
		typst:  typstService,
		binary: binaryService,
	}
}

// FindByID retrieves a file aggregate by ID.
func (s *Service) FindByID(ctx context.Context, fileID uuid.UUID) (domainFile.File, error) {
	if tf, err := s.typst.GetByID(ctx, fileID); err == nil {
		return tf, nil
	}
	if bf, err := s.binary.GetByID(ctx, fileID); err == nil {
		return bf, nil
	}
	return nil, fmt.Errorf("file not found: %s", fileID)
}

// Save persists a modified file aggregate.
func (s *Service) Save(ctx context.Context, f domainFile.File) error {
	switch file := f.(type) {
	case *domainFile.TypstFile:
		if err := s.typst.Save(ctx, file); err != nil {
			return fmt.Errorf("failed to save typst file: %w", err)
		}
		return nil
	case *domainFile.BinaryFile:
		if err := s.binary.Save(ctx, file); err != nil {
			return fmt.Errorf("failed to save binary file: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported file type: %T", f)
	}
}

// RenameFile mutates the file name on the domain aggregate and persists it.
func (s *Service) RenameFile(ctx context.Context, fileID uuid.UUID, newName string) error {
	f, err := s.FindByID(ctx, fileID)
	if err != nil {
		return err
	}

	if err := f.Rename(newName); err != nil {
		return fmt.Errorf("rename file %s: %w", fileID, err)
	}

	if err := s.Save(ctx, f); err != nil {
		return fmt.Errorf("save renamed file: %w", err)
	}
	return nil
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
