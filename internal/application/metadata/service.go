package metadata

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type Repository interface {
	FindEntriesByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domainEntry.Entry, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetMetadata retrieves project entries and builds the project metadata aggregate.
func (s *Service) GetMetadata(ctx context.Context, projectID uuid.UUID) (*domainMeta.Metadata, error) {
	entries, err := s.repo.FindEntriesByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata entries for project %s: %w", projectID, err)
	}

	meta, err := domainMeta.NewMetadata(projectID, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to build metadata aggregate for project %s: %w", projectID, err)
	}

	return meta, nil
}

// CreateMetadata creates a new Metadata aggregate from entries.
func (s *Service) CreateMetadata(projectID uuid.UUID, entries []*domainEntry.Entry) (*domainMeta.Metadata, error) {
	m, err := domainMeta.NewMetadata(projectID, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata aggregate: %w", err)
	}
	return m, nil
}
