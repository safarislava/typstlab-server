package file

import (
	"context"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type Repository interface {
	SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error
	SaveBinaryFile(ctx context.Context, f *domainFile.BinaryFile) error
	FindTypstFileByID(ctx context.Context, id uuid.UUID) (*domainFile.TypstFile, error)
	FindBinaryFileByID(ctx context.Context, id uuid.UUID) (*domainFile.BinaryFile, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error
	IsDeleted(ctx context.Context, id uuid.UUID) (bool, error)
}
