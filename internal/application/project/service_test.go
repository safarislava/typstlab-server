package project

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type mockRepository struct {
	saveFunc     func(ctx context.Context, p *domain.Project) error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Project, error)
}

func (m *mockRepository) Save(ctx context.Context, p *domain.Project) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, p)
	}
	return nil
}

func (m *mockRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("project not found")
}

type mockUserService struct {
	getByEmailFunc func(ctx context.Context, email string) (*domainUser.User, error)
	getByIDFunc    func(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

func (m *mockUserService) GetByEmail(ctx context.Context, email string) (*domainUser.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return nil, errors.New("user not found")
}

func (m *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("user not found")
}

func TestNewService(t *testing.T) {
	t.Parallel()

	svc := NewService(&mockRepository{}, &mockUserService{})
	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
}

func TestService_Create_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo, &mockUserService{})

	const testProjectName = "Test Project"
	projectID := uuid.New()
	req := CreateRequest{
		ID:     projectID,
		UserID: uuid.New(),
		Name:   testProjectName,
	}

	resp, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Name != testProjectName {
		t.Errorf("Expected Name %q, got %q", testProjectName, resp.Name)
	}

	if resp.ID != projectID {
		t.Errorf("Expected ID %s, got %s", projectID, resp.ID)
	}

	if resp.UpdatedAt.IsZero() {
		t.Error("Expected non-zero UpdatedAt")
	}
}

func TestService_Create_ValidationError(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo, &mockUserService{})

	req := CreateRequest{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Name:   "",
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create domain project") {
		t.Errorf("Expected error to contain validation message, got: %v", err)
	}
}

func TestService_Create_RepositoryError(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{
		saveFunc: func(ctx context.Context, p *domain.Project) error {
			return errors.New("database connection failed")
		},
	}
	svc := NewService(repo, &mockUserService{})

	req := CreateRequest{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Name:   "Valid Project",
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Expected repository error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to save project") {
		t.Errorf("Expected error to contain save error message, got: %v", err)
	}
}

func TestService_FindByID_Success(t *testing.T) {
	t.Parallel()

	p, err := domain.NewProject(uuid.New(), []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	repo := &mockRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			if id == p.ID() {
				return p, nil
			}
			return nil, errors.New("project not found")
		},
	}
	svc := NewService(repo, &mockUserService{})

	tp, err := svc.Get(context.Background(), p.ID())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(p, tp) {
		t.Fatalf("Expected project %v, got %v", p, tp)
	}
}

func TestService_FindByID_NotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo, &mockUserService{})

	_, err := svc.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestService_AddMember(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	newUserID := uuid.New()
	newUserEmail := "collaborator@example.com"
	newUser, _ := domainUser.NewUser(newUserID, newUserEmail, "hash", domainUser.RoleUser)

	p, _ := domain.NewProject(uuid.New(), []uuid.UUID{ownerID}, "Collab Project", time.Now())

	repo := &mockRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			if id == p.ID() {
				return p, nil
			}
			return nil, domain.ErrProjectNotFound
		},
		saveFunc: func(ctx context.Context, proj *domain.Project) error {
			return nil
		},
	}

	userSvc := &mockUserService{
		getByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			if email == newUserEmail {
				return newUser, nil
			}
			return nil, errors.New("user not found")
		},
	}

	svc := NewService(repo, userSvc)

	// Add member success
	member, err := svc.AddMember(context.Background(), p.ID(), newUserEmail)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if member.ID != newUserID || member.Email != newUserEmail {
		t.Errorf("Unexpected member: %v", member)
	}
	if !p.HasUser(newUserID) {
		t.Error("Project should have new user")
	}

	// Add same member again -> ErrUserAlreadyMember
	_, err = svc.AddMember(context.Background(), p.ID(), newUserEmail)
	if !errors.Is(err, domain.ErrUserAlreadyMember) {
		t.Errorf("Expected ErrUserAlreadyMember, got %v", err)
	}

	// User not found
	_, err = svc.AddMember(context.Background(), p.ID(), "nonexistent@example.com")
	if err == nil {
		t.Fatal("Expected error for non-existent user, got nil")
	}
}

func TestService_RemoveMember(t *testing.T) {
	t.Parallel()

	user1 := uuid.New()
	user2 := uuid.New()
	p, _ := domain.NewProject(uuid.New(), []uuid.UUID{user1, user2}, "Collab Project", time.Now())

	repo := &mockRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			return p, nil
		},
		saveFunc: func(ctx context.Context, proj *domain.Project) error {
			return nil
		},
	}

	svc := NewService(repo, &mockUserService{})

	if err := svc.RemoveMember(context.Background(), p.ID(), user2); err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}
	if p.HasUser(user2) {
		t.Error("Project should not have user2")
	}

	// Cannot remove last user
	if err := svc.RemoveMember(context.Background(), p.ID(), user1); !errors.Is(err, domain.ErrNoUsers) {
		t.Errorf("Expected ErrNoUsers, got %v", err)
	}
}

func TestService_ListMembers(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New()
	user1, _ := domainUser.NewUser(user1ID, "user1@example.com", "hash", domainUser.RoleUser)
	p, _ := domain.NewProject(uuid.New(), []uuid.UUID{user1ID}, "Test", time.Now())

	repo := &mockRepository{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
			return p, nil
		},
	}
	userSvc := &mockUserService{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
			if id == user1ID {
				return user1, nil
			}
			return nil, errors.New("not found")
		},
	}

	svc := NewService(repo, userSvc)
	members, err := svc.ListMembers(context.Background(), p.ID())
	if err != nil {
		t.Fatalf("Failed to list members: %v", err)
	}
	if len(members) != 1 || members[0].Email != "user1@example.com" {
		t.Errorf("Unexpected members list: %v", members)
	}
}
