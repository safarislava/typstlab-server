package typst

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type UploadRequest struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	State     []byte
	Blocks    []block.Block
}

type Repository interface {
	SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error
	FindTypstFileByID(ctx context.Context, id uuid.UUID) (*domainFile.TypstFile, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*domainFile.TypstFile, error) {
	f, err := domainFile.NewTypstFile(req.ID, req.ProjectID, req.Name, req.State, req.Blocks, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to upload typst file: %w", err)
	}

	if err := s.repo.SaveTypstFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save typst file: %w", err)
	}

	return f, nil
}

func (s *Service) Save(ctx context.Context, f *domainFile.TypstFile) error {
	if err := s.repo.SaveTypstFile(ctx, f); err != nil {
		return fmt.Errorf("failed to save typst file: %w", err)
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error) {
	f, err := s.repo.FindTypstFileByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}

	return f, nil
}
