package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

func TestNewFileRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	repo := NewFileRepository(pool)
	if repo == nil {
		t.Fatal("expected non-nil FileRepository")
	}
}

func TestFileRepository_ClosedPool_SaveAndFind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewFileRepository(pool)
	projectID := uuid.New()
	typstFileID := uuid.New()
	binaryFileID := uuid.New()

	tf, err := domainFile.NewTypstFile(typstFileID, projectID, "main.typ", []byte("state"), time.Now())
	if err != nil {
		t.Fatalf("failed to create typst file: %v", err)
	}

	bf, err := domainFile.NewBinaryFile(binaryFileID, projectID, "logo.png", []byte("img"), time.Now())
	if err != nil {
		t.Fatalf("failed to create binary file: %v", err)
	}

	if err := repo.SaveTypstFile(ctx, tf); err == nil {
		t.Error("expected error saving typst file with closed pool, got nil")
	}

	if err := repo.SaveBinaryFile(ctx, bf); err == nil {
		t.Error("expected error saving binary file with closed pool, got nil")
	}

	if _, err := repo.FindTypstFileByID(ctx, typstFileID); err == nil {
		t.Error("expected error finding typst file with closed pool, got nil")
	}

	if _, err := repo.FindBinaryFileByID(ctx, binaryFileID); err == nil {
		t.Error("expected error finding binary file with closed pool, got nil")
	}
}

func TestFileRepository_ClosedPool_DeleteAndEntries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewFileRepository(pool)
	projectID := uuid.New()
	fileID := uuid.New()

	if _, err := repo.FindByProjectID(ctx, projectID); err == nil {
		t.Error("expected error finding files by project with closed pool, got nil")
	}

	if err := repo.DeleteFile(ctx, fileID); err == nil {
		t.Error("expected error deleting file with closed pool, got nil")
	}

	if _, err := repo.FindEntriesByProjectID(ctx, projectID); err == nil {
		t.Error("expected error finding entries with closed pool, got nil")
	}
}
