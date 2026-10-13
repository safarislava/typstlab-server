package collaboration

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

// SyncService defines the contract required for synchronizing project files and metadata.
type SyncService interface {
	Sync(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error)
	ApplyFileChanges(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
}

// SyncInitRequest represents the application input for project sync initialization.
type SyncInitRequest struct {
	MetadataDelta       []byte
	MetadataStateVector []byte
	ContentVectors      map[uuid.UUID][]byte
}

// SyncInstruction represents a single sync instruction generated for a file.
type SyncInstruction struct {
	Action string
	FileID uuid.UUID
	Delta  []byte
}

// SyncInitResponse represents the application result of sync initialization.
type SyncInitResponse struct {
	MetadataDelta []byte
	Instructions  []SyncInstruction
}

// Service orchestrates real-time document synchronization and metadata updates.
type Service struct {
	syncService SyncService
}

// NewService creates a new collaboration Service.
func NewService(syncService SyncService) *Service {
	return &Service{
		syncService: syncService,
	}
}

// SyncInit coordinates initial project metadata sync and file content instruction generation.
func (s *Service) SyncInit(ctx context.Context, projectID uuid.UUID, req SyncInitRequest) (*SyncInitResponse, error) {
	appReq := &syncApp.Request{
		MetadataDelta:       req.MetadataDelta,
		MetadataStateVector: req.MetadataStateVector,
		ContentVectors:      req.ContentVectors,
	}

	resp, err := s.syncService.Sync(ctx, projectID, appReq)
	if err != nil {
		return nil, fmt.Errorf("sync init: %w", err)
	}

	instructions := make([]SyncInstruction, 0, len(resp.Instructions))
	for _, inst := range resp.Instructions {
		instructions = append(instructions, SyncInstruction{
			Action: string(inst.Action),
			FileID: inst.FileID,
			Delta:  inst.Delta,
		})
	}

	return &SyncInitResponse{
		MetadataDelta: resp.MetadataDelta,
		Instructions:  instructions,
	}, nil
}

// ApplyCRDTUpdate persists incremental CRDT file changes.
func (s *Service) ApplyCRDTUpdate(ctx context.Context, fileID uuid.UUID, delta []byte) error {
	if fileID == uuid.Nil {
		return errors.New("file_id is required")
	}
	if len(delta) == 0 {
		return errors.New("delta cannot be empty")
	}

	req := syncApp.ApplyFileChangesRequest{
		FileID: fileID,
		Delta:  delta,
	}
	if _, err := s.syncService.ApplyFileChanges(ctx, req); err != nil {
		return fmt.Errorf("apply crdt changes: %w", err)
	}
	return nil
}

// ApplyMetadataUpdate merges and persists incremental project metadata CRDT deltas.
func (s *Service) ApplyMetadataUpdate(ctx context.Context, projectID uuid.UUID, delta []byte) error {
	if len(delta) == 0 {
		return errors.New("delta cannot be empty")
	}

	req := &syncApp.Request{
		MetadataDelta: delta,
	}
	if _, err := s.syncService.Sync(ctx, projectID, req); err != nil {
		return fmt.Errorf("apply metadata update: %w", err)
	}
	return nil
}
