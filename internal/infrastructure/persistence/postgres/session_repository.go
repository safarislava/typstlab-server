package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/safarislava/typstlab-server/internal/domain/session"
	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// Save inserts or updates a session in PostgreSQL.
func (r *SessionRepository) Save(ctx context.Context, s domain.Session) error {
	query := `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			expires_at = EXCLUDED.expires_at
	`
	_, err := r.pool.Exec(ctx, query, s.Token().Value(), s.UserID(), s.ExpiresAt())
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

// FindByToken searches for a session by token value.
func (r *SessionRepository) FindByToken(ctx context.Context, t domainToken.Token) (domain.Session, error) {
	query := `
		SELECT token, user_id, expires_at
		FROM sessions
		WHERE token = $1
	`
	var (
		tokenVal  string
		userID    uuid.UUID
		expiresAt time.Time
	)

	err := r.pool.QueryRow(ctx, query, t.Value()).Scan(&tokenVal, &userID, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, fmt.Errorf("failed to find session by token: %w", err)
	}

	tok, err := domainToken.NewToken(tokenVal)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to construct token: %w", err)
	}

	sess, err := domain.NewSession(tok, userID, expiresAt)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to construct session: %w", err)
	}

	return sess, nil
}

// Delete removes a session by token.
func (r *SessionRepository) Delete(ctx context.Context, t domainToken.Token) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, t.Value())
	if err != nil {
		return fmt.Errorf("failed to delete session by token: %w", err)
	}
	return nil
}

// DeleteByUserID removes all active sessions for a given user ID.
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete sessions by user id: %w", err)
	}
	return nil
}
