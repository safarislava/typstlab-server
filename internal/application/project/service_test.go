package project

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
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
	return nil, nil
}

func TestNewService(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo)
	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
	if svc.repo != repo {
		t.Error("Service repo mismatch")
	}
}

func TestService_CreateProject_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo)

	const testProjectName = "Test Project"
	req := CreateProjectRequest{
		Name: testProjectName,
	}

	resp, err := svc.CreateProject(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Name != testProjectName {
		t.Errorf("Expected Name %q, got %q", testProjectName, resp.Name)
	}

	if resp.ID == uuid.Nil {
		t.Error("Expected valid generated ID, got uuid.Nil")
	}

	if resp.UpdatedAt.IsZero() {
		t.Error("Expected non-zero UpdatedAt")
	}
}

func TestService_CreateProject_ValidationError(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo)

	req := CreateProjectRequest{
		Name: "", // invalid name
	}

	_, err := svc.CreateProject(context.Background(), req)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create domain project") {
		t.Errorf("Expected error to contain validation message, got: %v", err)
	}
}

func TestService_CreateProject_SaveError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database connection down")
	repo := &mockRepository{
		saveFunc: func(ctx context.Context, p *domain.Project) error {
			return expectedErr
		},
	}
	svc := NewService(repo)

	req := CreateProjectRequest{
		Name: "Failing Save Project",
	}

	_, err := svc.CreateProject(context.Background(), req)
	if err == nil {
		t.Fatal("Expected repository save error, got nil")
	}

	if !errors.Is(err, expectedErr) && !strings.Contains(err.Error(), "failed to save project") {
		t.Errorf("Expected error containing 'failed to save project', got: %v", err)
	}
}
