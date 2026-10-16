package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

func TestNewProjectRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	repo := NewProjectRepository(pool)
	if repo == nil {
		t.Fatal("expected non-nil ProjectRepository")
	}
}

func TestProjectRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewProjectRepository(pool)

	proj, err := domain.NewProject(uuid.New(), []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := repo.Save(ctx, proj); err == nil {
		t.Error("expected error saving project with closed pool, got nil")
	}

	if _, err := repo.FindByID(ctx, proj.ID()); err == nil {
		t.Error("expected error finding project with closed pool, got nil")
	}

	if err := repo.Delete(ctx, proj.ID()); err == nil {
		t.Error("expected error deleting project with closed pool, got nil")
	}
}

func TestEncodePlan(t *testing.T) {
	m := pgtype.NewMap()

	// By default, []pgtype.UUID has a valid text encode plan
	planPgUUID := m.PlanEncode(0, pgtype.TextFormatCode, []pgtype.UUID{{Bytes: uuid.MustParse("bc4b4a5d-fe87-42bc-b994-293c482ad5d3"), Valid: true}})
	if planPgUUID == nil {
		t.Fatal("expected non-nil plan for []pgtype.UUID")
	}

	buf, err := planPgUUID.Encode([]pgtype.UUID{{Bytes: uuid.MustParse("bc4b4a5d-fe87-42bc-b994-293c482ad5d3"), Valid: true}}, nil)
	if err != nil {
		t.Fatalf("unexpected encode error for []pgtype.UUID: %v", err)
	}
	if string(buf) != "{bc4b4a5d-fe87-42bc-b994-293c482ad5d3}" {
		t.Errorf("unexpected encoded string: %s", string(buf))
	}

	// Registering []uuid.UUID allows pgx simple protocol / QueryExecModeExec to encode []uuid.UUID
	m.RegisterDefaultPgType([]uuid.UUID{}, "_uuid")
	planRegUUID := m.PlanEncode(0, pgtype.TextFormatCode, []uuid.UUID{uuid.MustParse("bc4b4a5d-fe87-42bc-b994-293c482ad5d3")})
	if planRegUUID == nil {
		t.Fatal("expected non-nil plan for registered []uuid.UUID")
	}
	bufReg, err := planRegUUID.Encode([]uuid.UUID{uuid.MustParse("bc4b4a5d-fe87-42bc-b994-293c482ad5d3")}, nil)
	if err != nil {
		t.Fatalf("unexpected encode error for registered []uuid.UUID: %v", err)
	}
	if string(bufReg) != "{bc4b4a5d-fe87-42bc-b994-293c482ad5d3}" {
		t.Errorf("unexpected encoded string: %s", string(bufReg))
	}
}
