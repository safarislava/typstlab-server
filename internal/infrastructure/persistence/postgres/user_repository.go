package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Save inserts or updates a user record in PostgreSQL.
func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			role = EXCLUDED.role,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.pool.Exec(ctx, query, u.ID(), u.Email(), u.PasswordHash(), string(u.Role()))
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

// FindByEmail searches for a user by their email address.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role
		FROM users
		WHERE email = $1
	`
	var (
		id           uuid.UUID
		emailVal     string
		passwordHash string
		roleStr      string
	)

	err := r.pool.QueryRow(ctx, query, email).Scan(&id, &emailVal, &passwordHash, &roleStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	u, err := domain.NewUser(id, emailVal, passwordHash, domain.Role(roleStr))
	if err != nil {
		return nil, fmt.Errorf("failed to construct user from db row: %w", err)
	}

	return u, nil
}

// FindByID searches for a user by their unique identifier.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role
		FROM users
		WHERE id = $1
	`
	var (
		idVal        uuid.UUID
		emailVal     string
		passwordHash string
		roleStr      string
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(&idVal, &emailVal, &passwordHash, &roleStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	u, err := domain.NewUser(idVal, emailVal, passwordHash, domain.Role(roleStr))
	if err != nil {
		return nil, fmt.Errorf("failed to construct user from db row: %w", err)
	}

	return u, nil
}
