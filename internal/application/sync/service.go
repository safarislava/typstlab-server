package sync

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	syncFile "github.com/safarislava/typstlab-server/internal/application/sync/file"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type Request struct {
	MetadataDelta       []byte
	MetadataStateVector []byte
	ContentVectors      map[uuid.UUID][]byte
}

type Action = syncFile.Action

const (
	ActionDownload     Action = syncFile.ActionDownload
	ActionUpload       Action = syncFile.ActionUpload
	ActionApplyChanges Action = syncFile.ActionApplyChanges
)

type Instruction = syncFile.Instruction

type ApplyFileChangesRequest = syncFile.ApplyFileChangesRequest

type Response struct {
	MetadataDelta []byte
	Instructions  []Instruction
}

type MetadataSyncer interface {
	SyncMetadata(
		ctx context.Context,
		projectID uuid.UUID,
		clientDelta []byte,
		clientStateVector []byte,
	) (metadataDelta []byte, updatedMeta *domainMeta.Metadata, err error)
}

type FileSyncer interface {
	ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	ApplyFileChanges(ctx context.Context, req syncFile.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
	ApplyMetadataMutations(ctx context.Context, serverFiles []domainFile.File, meta *domainMeta.Metadata) error
	GenerateContentInstructions(
		serverFiles []domainFile.File,
		meta *domainMeta.Metadata,
		contentVectors map[uuid.UUID][]byte,
	) ([]syncFile.Instruction, error)
}

type Service struct {
	metaSyncer MetadataSyncer
	fileSyncer FileSyncer
}

func NewService(metaSyncer MetadataSyncer, fileSyncer FileSyncer) *Service {
	return &Service{
		metaSyncer: metaSyncer,
		fileSyncer: fileSyncer,
	}
}

// Sync orchestrates project metadata sync, file updates and content delta calculation.
func (s *Service) Sync(ctx context.Context, projectID uuid.UUID, req *Request) (*Response, error) {
	metadataDelta, updatedMeta, err := s.metaSyncer.SyncMetadata(
		ctx,
		projectID,
		req.MetadataDelta,
		req.MetadataStateVector,
	)
	if err != nil {
		return nil, fmt.Errorf("metadata sync failed: %w", err)
	}

	serverFiles, err := s.fileSyncer.ListFilesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project files: %w", err)
	}

	if applyErr := s.fileSyncer.ApplyMetadataMutations(ctx, serverFiles, updatedMeta); applyErr != nil {
		return nil, fmt.Errorf("failed to apply metadata mutations to files: %w", applyErr)
	}

	instructions, err := s.fileSyncer.GenerateContentInstructions(serverFiles, updatedMeta, req.ContentVectors)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content instructions: %w", err)
	}

	return &Response{
		MetadataDelta: metadataDelta,
		Instructions:  instructions,
	}, nil
}

// ApplyFileChanges applies a CRDT delta to a specific Typst file and persists the updated state.
func (s *Service) ApplyFileChanges(ctx context.Context, req ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	f, err := s.fileSyncer.ApplyFileChanges(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to apply file changes: %w", err)
	}
	return f, nil
}
