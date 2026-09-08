package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

const (
	upsertFileQuery = `
		INSERT INTO files (id, project_id, name, type, state, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			state = EXCLUDED.state,
			updated_at = EXCLUDED.updated_at
	`
	findFileByIDAndTypeQuery = `
		SELECT project_id, name, state, updated_at
		FROM files
		WHERE id = $1 AND type = $2
	`
)

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

// SaveTypstFile inserts or updates a Typst file and its CRDT state in PostgreSQL.
func (r *FileRepository) SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error {
	_, err := r.pool.Exec(
		ctx,
		upsertFileQuery,
		f.ID(),
		f.ProjectID(),
		f.Name(),
		string(f.Type()),
		f.State(),
		f.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save typst file: %w", err)
	}
	return nil
}

// SaveBinaryFile inserts or updates a binary file in PostgreSQL.
func (r *FileRepository) SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error {
	_, err := r.pool.Exec(
		ctx,
		upsertFileQuery,
		f.ID(),
		f.ProjectID(),
		f.Name(),
		string(f.Type()),
		f.Content(),
		f.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save binary file: %w", err)
	}
	return nil
}

// FindTypstFileByID retrieves a Typst file by its ID.
func (r *FileRepository) FindTypstFileByID(ctx context.Context, id uuid.UUID) (*domainFile.TypstFile, error) {
	var (
		projectID uuid.UUID
		name      string
		state     []byte
		updatedAt time.Time
	)

	err := r.pool.QueryRow(ctx, findFileByIDAndTypeQuery, id, string(domainFile.TypeTypst)).Scan(
		&projectID,
		&name,
		&state,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainFile.ErrTypstFileNotFound
		}
		return nil, fmt.Errorf("failed to find typst file by id: %w", err)
	}

	f, err := domainFile.NewTypstFile(id, projectID, name, state, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to construct typst file: %w", err)
	}

	return f, nil
}

// FindBinaryFileByID retrieves a binary file by its ID.
func (r *FileRepository) FindBinaryFileByID(ctx context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
	var (
		projectID uuid.UUID
		name      string
		content   []byte
		updatedAt time.Time
	)

	err := r.pool.QueryRow(ctx, findFileByIDAndTypeQuery, id, string(domainFile.TypeBinary)).Scan(
		&projectID,
		&name,
		&content,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainFile.ErrBinaryFileNotFound
		}
		return nil, fmt.Errorf("failed to find binary file by id: %w", err)
	}

	f, err := domainFile.NewBinaryFile(id, projectID, name, content, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to construct binary file: %w", err)
	}

	return f, nil
}

// FindByProjectID retrieves all files belonging to a project.
func (r *FileRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	query := `
		SELECT id, project_id, name, type, state, updated_at
		FROM files
		WHERE project_id = $1
	`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query files for project %s: %w", projectID, err)
	}
	defer rows.Close()

	var result []domainFile.File
	for rows.Next() {
		var (
			id        uuid.UUID
			pID       uuid.UUID
			name      string
			typeStr   string
			state     []byte
			updatedAt time.Time
		)

		if scanErr := rows.Scan(&id, &pID, &name, &typeStr, &state, &updatedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan file row: %w", scanErr)
		}

		fileObj, mapErr := mapFileRow(id, pID, name, typeStr, state, updatedAt)
		if mapErr != nil {
			return nil, mapErr
		}
		if fileObj != nil {
			result = append(result, fileObj)
		}
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("error iterating file rows: %w", rowsErr)
	}

	return result, nil
}

func mapFileRow(id, pID uuid.UUID, name, typeStr string, state []byte, updatedAt time.Time) (domainFile.File, error) {
	switch domainFile.Type(typeStr) {
	case domainFile.TypeTypst:
		tf, err := domainFile.NewTypstFile(id, pID, name, state, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to construct typst file %s: %w", id, err)
		}
		return tf, nil
	case domainFile.TypeBinary:
		bf, err := domainFile.NewBinaryFile(id, pID, name, state, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to construct binary file %s: %w", id, err)
		}
		return bf, nil
	default:
		return nil, nil
	}
}

// DeleteFile deletes a file record from PostgreSQL.
func (r *FileRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM files WHERE id = $1`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domainFile.ErrFileNotFound
	}
	return nil
}

// FindEntriesByProjectID retrieves metadata entries for all files of a project.
func (r *FileRepository) FindEntriesByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domainEntry.Entry, error) {
	query := `
		SELECT id, name, type, updated_at
		FROM files
		WHERE project_id = $1
	`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata entries for project %s: %w", projectID, err)
	}
	defer rows.Close()

	var entries []*domainEntry.Entry
	for rows.Next() {
		var (
			id        uuid.UUID
			name      string
			typeStr   string
			updatedAt time.Time
		)

		if scanErr := rows.Scan(&id, &name, &typeStr, &updatedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan entry row: %w", scanErr)
		}

		entry, newErr := domainEntry.NewEntry(id, name, domainFile.Type(typeStr), false, updatedAt)
		if newErr != nil {
			return nil, fmt.Errorf("failed to create entry %s: %w", id, newErr)
		}
		entries = append(entries, entry)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("error iterating entry rows: %w", rowsErr)
	}

	return entries, nil
}
