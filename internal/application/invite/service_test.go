package invite

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domainInvite "github.com/safarislava/typstlab-server/internal/domain/invite"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
)

type mockProjectRepository struct {
	findByIDFunc func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error)
	saveFunc     func(ctx context.Context, p *domainProject.Project) error
}

func (m *mockProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domainProject.ErrProjectNotFound
}

func (m *mockProjectRepository) Save(ctx context.Context, p *domainProject.Project) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, p)
	}
	return nil
}

type inMemoryInviteRepo struct {
	invites []*domainInvite.Invite
}

func (r *inMemoryInviteRepo) Save(_ context.Context, inv *domainInvite.Invite) error {
	r.invites = append(r.invites, inv)
	return nil
}

func (r *inMemoryInviteRepo) FindByCode(_ context.Context, code string) (*domainInvite.Invite, error) {
	for _, inv := range r.invites {
		if inv.Code() == code {
			return inv, nil
		}
	}
	return nil, domainInvite.ErrInviteNotFound
}

func (r *inMemoryInviteRepo) FindByProjectID(_ context.Context, _ uuid.UUID) ([]*domainInvite.Invite, error) {
	return r.invites, nil
}

func (r *inMemoryInviteRepo) FindByID(_ context.Context, id uuid.UUID) (*domainInvite.Invite, error) {
	for _, inv := range r.invites {
		if inv.ID() == id {
			return inv, nil
		}
	}
	return nil, domainInvite.ErrInviteNotFound
}

func (r *inMemoryInviteRepo) Delete(_ context.Context, id uuid.UUID) error {
	for i, inv := range r.invites {
		if inv.ID() == id {
			r.invites = append(r.invites[:i], r.invites[i+1:]...)
			return nil
		}
	}
	return domainInvite.ErrInviteNotFound
}

func TestInviteService_Flow(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	joinerID := uuid.New()
	p, _ := domainProject.NewProject(uuid.New(), []uuid.UUID{ownerID}, "Invite Project", time.Now())

	inviteRepo := &inMemoryInviteRepo{}
	projectRepo := &mockProjectRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
			if id == p.ID() {
				return p, nil
			}
			return nil, domainProject.ErrProjectNotFound
		},
	}

	svc := NewService(projectRepo, inviteRepo)

	// Create invite
	inv, err := svc.CreateInvite(context.Background(), p.ID(), ownerID, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to create invite: %v", err)
	}
	if inv.Code() == "" {
		t.Error("Expected non-empty invite code")
	}

	// List invites
	invites, err := svc.ListInvites(context.Background(), p.ID())
	if err != nil || len(invites) != 1 {
		t.Fatalf("Expected 1 invite, got %d, err: %v", len(invites), err)
	}

	// Accept invite with joiner
	joinedProj, err := svc.AcceptInvite(context.Background(), inv.Code(), joinerID)
	if err != nil {
		t.Fatalf("Failed to accept invite: %v", err)
	}
	if !joinedProj.HasUser(joinerID) {
		t.Error("Expected project to contain joiner")
	}

	// Revoke invite
	if err := svc.RevokeInvite(context.Background(), p.ID(), inv.ID()); err != nil {
		t.Fatalf("Failed to revoke invite: %v", err)
	}
	if len(inviteRepo.invites) != 0 {
		t.Error("Expected invite to be revoked")
	}
}
