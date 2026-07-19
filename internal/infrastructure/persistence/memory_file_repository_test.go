package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

func TestMemoryFileRepository_SaveAndFindTypst(t *testing.T) {
	t.Parallel()

	repo := NewMemoryFileRepository()
	ctx := context.Background()

	projectID := uuid.New()
	fileID := uuid.New()

	tf, err := domainFile.NewTypstFile(fileID, projectID, "doc.typ", []byte("initial-state"), []block.Block(nil), time.Now())
	if err != nil {
		t.Fatalf("failed to create typst file: %v", err)
	}

	if saveErr := repo.SaveTypstFile(ctx, tf); saveErr != nil {
		t.Fatalf("failed to save typst file: %v", saveErr)
	}

	tfFound, findErr := repo.FindTypstFileByID(ctx, fileID)
	if findErr != nil {
		t.Fatalf("failed to find typst file: %v", findErr)
	}
	if tfFound.ID() != fileID {
		t.Errorf("expected id %v, got %v", fileID, tfFound.ID())
	}
}

func TestMemoryFileRepository_SaveAndFindBinary(t *testing.T) {
	t.Parallel()

	repo := NewMemoryFileRepository()
	ctx := context.Background()

	projectID := uuid.New()
	fileID := uuid.New()

	bf, err := domainFile.NewBinaryFile(fileID, projectID, "img.png", []byte{1, 2, 3}, time.Now())
	if err != nil {
		t.Fatalf("failed to create binary file: %v", err)
	}

	if saveErr := repo.SaveBinaryFile(ctx, bf); saveErr != nil {
		t.Fatalf("failed to save binary file: %v", saveErr)
	}

	bfFound, findErr := repo.FindBinaryFileByID(ctx, fileID)
	if findErr != nil {
		t.Fatalf("failed to find binary file: %v", findErr)
	}
	if bfFound.ID() != fileID {
		t.Errorf("expected id %v, got %v", fileID, bfFound.ID())
	}
}

func TestMemoryFileRepository_FindByProjectID(t *testing.T) {
	t.Parallel()

	repo := NewMemoryFileRepository()
	ctx := context.Background()

	projectID := uuid.New()
	fileID1 := uuid.New()
	fileID2 := uuid.New()

	tf, err1 := domainFile.NewTypstFile(fileID1, projectID, "doc.typ", []byte("initial-state"), []block.Block(nil), time.Now())
	if err1 != nil {
		t.Fatalf("failed to create typst file: %v", err1)
	}
	if saveErr := repo.SaveTypstFile(ctx, tf); saveErr != nil {
		t.Fatalf("failed to save typst file: %v", saveErr)
	}

	bf, err2 := domainFile.NewBinaryFile(fileID2, projectID, "img.png", []byte{1, 2, 3}, time.Now())
	if err2 != nil {
		t.Fatalf("failed to create binary file: %v", err2)
	}
	if saveErr := repo.SaveBinaryFile(ctx, bf); saveErr != nil {
		t.Fatalf("failed to save binary file: %v", saveErr)
	}

	files, findErr := repo.FindByProjectID(ctx, projectID)
	if findErr != nil {
		t.Fatalf("failed to find by project id: %v", findErr)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestMemoryFileRepository_Delete(t *testing.T) {
	t.Parallel()

	repo := NewMemoryFileRepository()
	ctx := context.Background()

	projectID := uuid.New()
	fileID := uuid.New()

	tf, err := domainFile.NewTypstFile(fileID, projectID, "doc.typ", []byte("initial-state"), []block.Block(nil), time.Now())
	if err != nil {
		t.Fatalf("failed to create typst file: %v", err)
	}
	if saveErr := repo.SaveTypstFile(ctx, tf); saveErr != nil {
		t.Fatalf("failed to save typst file: %v", saveErr)
	}

	if delErr := repo.DeleteFile(ctx, fileID); delErr != nil {
		t.Fatalf("failed to delete file: %v", delErr)
	}

	_, findErr := repo.FindTypstFileByID(ctx, fileID)
	if findErr == nil {
		t.Error("expected error finding deleted file, got nil")
	}

	// Delete non-existent file
	delErr2 := repo.DeleteFile(ctx, fileID)
	if delErr2 == nil {
		t.Error("expected error deleting non-existent file, got nil")
	}
}
