package file

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type CreateTypstFileRequest struct {
	ProjectID uuid.UUID
	Name      string
}

type CreateBinaryFileRequest struct {
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
	CreateTypstFile(ctx context.Context, req CreateTypstFileRequest) (*domainFile.TypstFile, error)
	CreateBinaryFile(ctx context.Context, req CreateBinaryFileRequest) (*domainFile.BinaryFile, error)
	GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	GetBinaryFile(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error)
	ApplyFileChanges(ctx context.Context, req ApplyFileChangesRequest) (*domainFile.TypstFile, error)
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
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

func (s *Service) CreateTypstFile(ctx context.Context, req CreateTypstFileRequest) (*domainFile.TypstFile, error) {
	var initialState []byte
	f, err := domainFile.NewTypstFile(uuid.New(), req.ProjectID, req.Name, initialState, []block.Block(nil), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create typst file: %w", err)
	}

	if err := s.repo.SaveTypstFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save typst file: %w", err)
	}

	return f, nil
}

func (s *Service) CreateBinaryFile(ctx context.Context, req CreateBinaryFileRequest) (*domainFile.BinaryFile, error) {
	f, err := domainFile.NewBinaryFile(uuid.New(), req.ProjectID, req.Name, req.Content, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create typst file: %w", err)
	}

	if err := s.repo.SaveBinaryFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save typst file: %w", err)
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
