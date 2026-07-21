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

type ApplyFileChangesRequest struct {
	FileID uuid.UUID
	Delta  []byte
}

type Response struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	UpdatedAt time.Time
}

type UseCase interface {
	UploadTypstFile(ctx context.Context, req *UploadTypstFileRequest) (*domainFile.TypstFile, error)
	UploadBinaryFile(ctx context.Context, req *UploadBinaryFileRequest) (*domainFile.BinaryFile, error)
	GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	GetBinaryFile(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error)
	ApplyFileChanges(ctx context.Context, req ApplyFileChangesRequest) (*domainFile.TypstFile, error)
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
	ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
}

type Service struct {
	repo   Repository
	merger Merger
}

func NewService(repo Repository, merger Merger) UseCase {
	return &Service{
		repo:   repo,
		merger: merger,
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
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}

	return f, nil
}

func (s *Service) ApplyFileChanges(ctx context.Context, req ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	f, err := s.repo.FindTypstFileByID(ctx, req.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}

	state, blocks, err := s.merger.MergeFile(f.State(), req.Delta)
	if err != nil {
		return nil, fmt.Errorf("failed to merge file delta: %w", err)
	}

	if err := f.UpdateState(state, blocks); err != nil {
		return nil, fmt.Errorf("failed to update typst file aggregate state: %w", err)
	}

	if err := s.repo.SaveTypstFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save updated typst file: %w", err)
	}

	return f, nil
}

func (s *Service) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	if err := s.repo.DeleteFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete typst file: %w", err)
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
