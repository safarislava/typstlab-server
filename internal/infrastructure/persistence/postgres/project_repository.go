package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// Save inserts or updates a project and synchronizes its members in a transaction.
func (r *ProjectRepository) Save(ctx context.Context, p *domain.Project) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	upsertProjectQuery := `
		INSERT INTO projects (id, name, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			updated_at = EXCLUDED.updated_at
	`
	_, err = tx.Exec(ctx, upsertProjectQuery, p.ID(), p.Name(), p.UpdatedAt())
	if err != nil {
		return fmt.Errorf("failed to upsert project: %w", err)
	}

	insertMemberQuery := `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`
	for _, uid := range p.UserIDs() {
		_, err = tx.Exec(ctx, insertMemberQuery, p.ID(), uid)
		if err != nil {
			return fmt.Errorf("failed to insert project member %s: %w", uid, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit project transaction: %w", err)
	}

	return nil
}

// FindByID retrieves a project and its associated members by project ID.
func (r *ProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	projectQuery := `
		SELECT name, updated_at
		FROM projects
		WHERE id = $1
	`
	var (
		name      string
		updatedAt time.Time
	)

	err := r.pool.QueryRow(ctx, projectQuery, id).Scan(&name, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	membersQuery := `
		SELECT user_id
		FROM project_members
		WHERE project_id = $1
	`
	membersRows, queryErr := r.pool.Query(ctx, membersQuery, id)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to query project members: %w", queryErr)
	}
	defer membersRows.Close()

	var userIDs []uuid.UUID
	for membersRows.Next() {
		var uid uuid.UUID
		if scanErr := membersRows.Scan(&uid); scanErr != nil {
			return nil, fmt.Errorf("failed to scan member user_id: %w", scanErr)
		}
		userIDs = append(userIDs, uid)
	}

	if rowsErr := membersRows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("error iterating project members: %w", rowsErr)
	}

	p, err := domain.NewProject(id, userIDs, name, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to construct project from db: %w", err)
	}

	return p, nil
}

// Delete removes a project and cascades deletion to members and files.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}
