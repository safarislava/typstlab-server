package invite

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	domainInvite "github.com/safarislava/typstlab-server/internal/domain/invite"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
)

type ProjectRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domainProject.Project, error)
	Save(ctx context.Context, p *domainProject.Project) error
}

type Repository interface {
	Save(ctx context.Context, inv *domainInvite.Invite) error
	FindByCode(ctx context.Context, code string) (*domainInvite.Invite, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domainInvite.Invite, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domainInvite.Invite, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	projectRepo ProjectRepository
	inviteRepo  Repository
}

func NewService(projectRepo ProjectRepository, inviteRepo Repository) *Service {
	return &Service{
		projectRepo: projectRepo,
		inviteRepo:  inviteRepo,
	}
}

func (s *Service) CreateInvite(ctx context.Context, projectID, createdBy uuid.UUID, expiresAt time.Time) (*domainInvite.Invite, error) {
	p, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if !p.HasUser(createdBy) {
		return nil, domainProject.ErrUserNotInProject
	}

	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	codeBytes := make([]byte, 16)
	if _, readErr := rand.Read(codeBytes); readErr != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", readErr)
	}
	code := hex.EncodeToString(codeBytes)

	inv, err := domainInvite.NewInvite(uuid.New(), projectID, createdBy, code, expiresAt, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create domain invite: %w", err)
	}

	if err := s.inviteRepo.Save(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to save invite: %w", err)
	}

	return inv, nil
}

func (s *Service) ListInvites(ctx context.Context, projectID uuid.UUID) ([]*domainInvite.Invite, error) {
	p, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	invites, err := s.inviteRepo.FindByProjectID(ctx, p.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to find invites by project: %w", err)
	}

	return invites, nil
}

func (s *Service) RevokeInvite(ctx context.Context, projectID, inviteID uuid.UUID) error {
	inv, err := s.inviteRepo.FindByID(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("invite not found: %w", err)
	}

	if inv.ProjectID() != projectID {
		return domainInvite.ErrInviteNotFound
	}

	if err := s.inviteRepo.Delete(ctx, inviteID); err != nil {
		return fmt.Errorf("failed to delete invite: %w", err)
	}

	return nil
}

func (s *Service) AcceptInvite(ctx context.Context, code string, userID uuid.UUID) (*domainProject.Project, error) {
	inv, err := s.inviteRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("find invite: %w", err)
	}

	if inv.IsExpired(time.Now()) {
		return nil, domainInvite.ErrInviteExpired
	}

	p, err := s.projectRepo.FindByID(ctx, inv.ProjectID())
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}

	if p.HasUser(userID) {
		return p, nil
	}

	if err := p.AddUser(userID); err != nil {
		return nil, fmt.Errorf("add user to project: %w", err)
	}

	if err := s.projectRepo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("save project: %w", err)
	}

	return p, nil
}
