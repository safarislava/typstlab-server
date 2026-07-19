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

type ApplyBlockChangesRequest struct {
	FileID  uuid.UUID
	BlockID uuid.UUID
	Delta   []byte
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
	ApplyBlockChanges(ctx context.Context, req ApplyBlockChangesRequest) (*domainFile.TypstFile, error)
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
	f, err := domainFile.NewTypstFile(uuid.New(), req.ProjectID, req.Name, []block.Block(nil), time.Now())
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

func (s *Service) ApplyBlockChanges(ctx context.Context, req ApplyBlockChangesRequest) (*domainFile.TypstFile, error) {
	f, err := s.repo.FindTypstFileByID(ctx, req.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}

	blocks := f.Blocks()
	found := false
	for i, b := range blocks {
		if b.ID() != req.BlockID {
			continue
		}

		state, content, mergeErr := s.merger.MergeBlock(b.State(), req.Delta)
		if mergeErr != nil {
			return nil, fmt.Errorf("failed to merge block: %w", mergeErr)
		}

		newBlock, blockErr := block.NewBlock(b.ID(), b.Name(), state, content)
		if blockErr != nil {
			return nil, fmt.Errorf("failed to create new block: %w", blockErr)
		}

		blocks[i] = newBlock
		found = true
		break
	}

	if !found {
		return nil, fmt.Errorf("block %s not found in file %s", req.BlockID, req.FileID)
	}

	newFile, fileErr := domainFile.NewTypstFile(f.ID(), f.ProjectID(), f.Name(), blocks, time.Now())
	if fileErr != nil {
		return nil, fmt.Errorf("failed to create new typst file: %w", fileErr)
	}

	return newFile, nil
}

func (s *Service) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	if err := s.repo.DeleteFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete typst file: %w", err)
	}

	return nil
}
