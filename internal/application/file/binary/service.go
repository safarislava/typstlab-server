package binary

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type UploadRequest struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	Content   []byte
}

type Repository interface {
	SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error
	FindBinaryFileByID(ctx context.Context, id uuid.UUID) (*domainFile.BinaryFile, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*domainFile.BinaryFile, error) {
	f, err := domainFile.NewBinaryFile(req.ID, req.ProjectID, req.Name, req.Content, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to upload binary file: %w", err)
	}

	if err := s.repo.SaveBinaryFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save binary file: %w", err)
	}

	return f, nil
}

func (s *Service) Save(ctx context.Context, f *domainFile.BinaryFile) error {
	if err := s.repo.SaveBinaryFile(ctx, f); err != nil {
		return fmt.Errorf("failed to save binary file: %w", err)
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error) {
	f, err := s.repo.FindBinaryFileByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find binary file: %w", err)
	}

	return f, nil
}
