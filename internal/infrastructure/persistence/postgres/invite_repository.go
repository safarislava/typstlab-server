package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/safarislava/typstlab-server/internal/domain/invite"
)

type InviteRepository struct {
	pool *pgxpool.Pool
}

func NewInviteRepository(pool *pgxpool.Pool) *InviteRepository {
	return &InviteRepository{pool: pool}
}

func (r *InviteRepository) Save(ctx context.Context, inv *domain.Invite) error {
	query := `
		INSERT INTO project_invites (id, project_id, code, created_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			expires_at = EXCLUDED.expires_at
	`
	_, err := r.pool.Exec(ctx, query, inv.ID(), inv.ProjectID(), inv.Code(), inv.CreatedBy(), inv.ExpiresAt(), inv.CreatedAt())
	if err != nil {
		return fmt.Errorf("failed to save invite: %w", err)
	}
	return nil
}

func (r *InviteRepository) FindByCode(ctx context.Context, code string) (*domain.Invite, error) {
	query := `
		SELECT id, project_id, created_by, expires_at, created_at
		FROM project_invites
		WHERE code = $1
	`
	var (
		id        uuid.UUID
		projectID uuid.UUID
		createdBy uuid.UUID
		expiresAt time.Time
		createdAt time.Time
	)
	err := r.pool.QueryRow(ctx, query, code).Scan(&id, &projectID, &createdBy, &expiresAt, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInviteNotFound
		}
		return nil, fmt.Errorf("failed to find invite by code: %w", err)
	}

	inv, err := domain.NewInvite(id, projectID, createdBy, code, expiresAt, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct invite: %w", err)
	}
	return inv, nil
}

func (r *InviteRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.Invite, error) {
	query := `
		SELECT id, code, created_by, expires_at, created_at
		FROM project_invites
		WHERE project_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query project invites: %w", err)
	}
	defer rows.Close()

	var invites []*domain.Invite
	for rows.Next() {
		var (
			id        uuid.UUID
			code      string
			createdBy uuid.UUID
			expiresAt time.Time
			createdAt time.Time
		)
		if err := rows.Scan(&id, &code, &createdBy, &expiresAt, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan invite row: %w", err)
		}
		inv, err := domain.NewInvite(id, projectID, createdBy, code, expiresAt, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct invite: %w", err)
		}
		invites = append(invites, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invite rows: %w", err)
	}

	return invites, nil
}

func (r *InviteRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Invite, error) {
	query := `
		SELECT project_id, code, created_by, expires_at, created_at
		FROM project_invites
		WHERE id = $1
	`
	var (
		projectID uuid.UUID
		code      string
		createdBy uuid.UUID
		expiresAt time.Time
		createdAt time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(&projectID, &code, &createdBy, &expiresAt, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInviteNotFound
		}
		return nil, fmt.Errorf("failed to find invite by id: %w", err)
	}

	inv, err := domain.NewInvite(id, projectID, createdBy, code, expiresAt, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct invite: %w", err)
	}
	return inv, nil
}

func (r *InviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM project_invites WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete invite: %w", err)
	}
	return nil
}
