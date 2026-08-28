package metadata

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type Manager interface {
	GetMetadata(ctx context.Context, projectID uuid.UUID) (*domainMeta.Metadata, error)
}

type Syncer interface {
	SyncMetadata(
		projectID uuid.UUID,
		currentMeta *domainMeta.Metadata,
		clientDelta []byte,
		clientStateVector []byte,
	) (metadataDelta []byte, meta *domainMeta.Metadata, err error)
}

type Service struct {
	metaManager Manager
	syncer      Syncer
}

func NewService(metaManager Manager, syncer Syncer) *Service {
	return &Service{
		metaManager: metaManager,
		syncer:      syncer,
	}
}

// SyncMetadata fetches current metadata, synchronizes with client delta, and returns delta & updated aggregate.
func (s *Service) SyncMetadata(
	ctx context.Context,
	projectID uuid.UUID,
	clientDelta []byte,
	clientStateVector []byte,
) (metadataDelta []byte, updatedMeta *domainMeta.Metadata, err error) {
	currentMeta, err := s.metaManager.GetMetadata(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch current project metadata: %w", err)
	}

	delta, meta, err := s.syncer.SyncMetadata(projectID, currentMeta, clientDelta, clientStateVector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sync project metadata CRDT: %w", err)
	}

	return delta, meta, nil
}
