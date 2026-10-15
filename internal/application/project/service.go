package project

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type CreateRequest struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Name   string
}

type CreateResponse struct {
	ID        uuid.UUID
	Name      string
	UpdatedAt time.Time
}

type MemberDTO struct {
	ID    uuid.UUID
	Email string
}

type Repository interface {
	Save(ctx context.Context, p *domain.Project) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error)
}

type UserService interface {
	GetByEmail(ctx context.Context, email string) (*domainUser.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

type Service struct {
	repo    Repository
	userSvc UserService
}

func NewService(repo Repository, userSvc UserService) *Service {
	return &Service{
		repo:    repo,
		userSvc: userSvc,
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	p, err := domain.NewProject(req.ID, []uuid.UUID{req.UserID}, req.Name, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create domain project: %w", err)
	}

	if err := s.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return &CreateResponse{
		ID:        p.ID(),
		Name:      p.Name(),
		UpdatedAt: p.UpdatedAt(),
	}, nil
}

func (s *Service) Get(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	p, err := s.repo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	return p, nil
}

func (s *Service) AddMember(ctx context.Context, projectID uuid.UUID, email string) (*MemberDTO, error) {
	u, err := s.userSvc.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	p, err := s.repo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if p.HasUser(u.ID()) {
		return nil, domain.ErrUserAlreadyMember
	}

	if err := p.AddUser(u.ID()); err != nil {
		return nil, fmt.Errorf("failed to add user to project: %w", err)
	}

	if err := s.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return &MemberDTO{
		ID:    u.ID(),
		Email: u.Email(),
	}, nil
}

func (s *Service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	p, err := s.repo.FindByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	if err := p.RemoveUser(userID); err != nil {
		return fmt.Errorf("remove user: %w", err)
	}

	if err := s.repo.Save(ctx, p); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

func (s *Service) ListMembers(ctx context.Context, projectID uuid.UUID) ([]MemberDTO, error) {
	p, err := s.repo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	userIDs := p.UserIDs()
	members := make([]MemberDTO, 0, len(userIDs))
	for _, uid := range userIDs {
		dto := MemberDTO{ID: uid}
		if u, err := s.userSvc.GetByID(ctx, uid); err == nil && u != nil {
			dto.Email = u.Email()
		}
		members = append(members, dto)
	}

	return members, nil
}
