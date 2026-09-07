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

func (s *Service) RenameFile(ctx context.Context, fileID uuid.UUID, newName string) error {
	tf, errTypst := s.typst.GetByID(ctx, fileID)
	if errTypst == nil {
		if err := tf.Rename(newName); err != nil {
			return fmt.Errorf("failed to rename typst file %s: %w", fileID, err)
		}
		if err := s.typst.Save(ctx, tf); err != nil {
			return fmt.Errorf("failed to save renamed typst file %s: %w", fileID, err)
		}
		return nil
	}

	bf, errBinary := s.binary.GetByID(ctx, fileID)
	if errBinary == nil {
		if err := bf.Rename(newName); err != nil {
			return fmt.Errorf("failed to rename binary file %s: %w", fileID, err)
		}
		if err := s.binary.Save(ctx, bf); err != nil {
			return fmt.Errorf("failed to save renamed binary file %s: %w", fileID, err)
		}
		return nil
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
