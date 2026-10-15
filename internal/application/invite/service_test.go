package invite

import (
	"context"
	"errors"
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

func TestInviteService_CreateInvite_Errors(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	p, _ := domainProject.NewProject(uuid.New(), []uuid.UUID{ownerID}, "Proj", time.Now())

	t.Run("ProjectNotFound", func(t *testing.T) {
		t.Parallel()
		svc := NewService(&mockProjectRepository{}, &inMemoryInviteRepo{})
		_, err := svc.CreateInvite(context.Background(), uuid.New(), ownerID, time.Time{})
		if err == nil {
			t.Error("Expected error when project not found")
		}
	})

	t.Run("UserNotInProject", func(t *testing.T) {
		t.Parallel()
		projectRepo := &mockProjectRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
				return p, nil
			},
		}
		svc := NewService(projectRepo, &inMemoryInviteRepo{})
		_, err := svc.CreateInvite(context.Background(), p.ID(), uuid.New(), time.Time{})
		if !errors.Is(err, domainProject.ErrUserNotInProject) {
			t.Errorf("Expected ErrUserNotInProject, got %v", err)
		}
	})

	t.Run("DefaultExpiration", func(t *testing.T) {
		t.Parallel()
		projectRepo := &mockProjectRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
				return p, nil
			},
		}
		svc := NewService(projectRepo, &inMemoryInviteRepo{})
		inv, err := svc.CreateInvite(context.Background(), p.ID(), ownerID, time.Time{})
		if err != nil {
			t.Fatalf("Failed to create invite: %v", err)
		}
		if inv.ExpiresAt().Before(time.Now().Add(6 * 24 * time.Hour)) {
			t.Errorf("Expected default expiration in ~7 days, got %v", inv.ExpiresAt())
		}
	})
}

func TestInviteService_RevokeAndAccept_Errors(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	otherUserID := uuid.New()
	p, _ := domainProject.NewProject(uuid.New(), []uuid.UUID{ownerID}, "Proj", time.Now())
	invID := uuid.New()
	code := "revokecode"
	expiredInv, _ := domainInvite.NewInvite(invID, p.ID(), ownerID, code, time.Now().Add(-1*time.Hour), time.Now().Add(-2*time.Hour))

	repo := &inMemoryInviteRepo{invites: []*domainInvite.Invite{expiredInv}}
	projectRepo := &mockProjectRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
			if id == p.ID() {
				return p, nil
			}
			return nil, domainProject.ErrProjectNotFound
		},
	}
	svc := NewService(projectRepo, repo)

	t.Run("RevokeWrongProject", func(t *testing.T) {
		t.Parallel()
		err := svc.RevokeInvite(context.Background(), uuid.New(), invID)
		if !errors.Is(err, domainInvite.ErrInviteNotFound) {
			t.Errorf("Expected ErrInviteNotFound, got %v", err)
		}
	})

	t.Run("AcceptExpired", func(t *testing.T) {
		t.Parallel()
		_, err := svc.AcceptInvite(context.Background(), code, otherUserID)
		if !errors.Is(err, domainInvite.ErrInviteExpired) {
			t.Errorf("Expected ErrInviteExpired, got %v", err)
		}
	})

	t.Run("AcceptAlreadyMember", func(t *testing.T) {
		t.Parallel()
		activeInv, _ := domainInvite.NewInvite(uuid.New(), p.ID(), ownerID, "activecode", time.Now().Add(24*time.Hour), time.Now())
		activeRepo := &inMemoryInviteRepo{invites: []*domainInvite.Invite{activeInv}}
		activeSvc := NewService(projectRepo, activeRepo)
		proj, err := activeSvc.AcceptInvite(context.Background(), "activecode", ownerID)
		if err != nil {
			t.Fatalf("Expected nil error for existing member, got %v", err)
		}
		if proj == nil {
			t.Fatal("Expected project")
		}
	})
}
