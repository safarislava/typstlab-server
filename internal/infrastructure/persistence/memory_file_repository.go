package persistence

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type MemoryFileRepository struct {
	mu          sync.RWMutex
	typstFiles  map[uuid.UUID]*domainFile.TypstFile
	binaryFiles map[uuid.UUID]*domainFile.BinaryFile
}

func NewMemoryFileRepository() *MemoryFileRepository {
	return &MemoryFileRepository{
		typstFiles:  make(map[uuid.UUID]*domainFile.TypstFile),
		binaryFiles: make(map[uuid.UUID]*domainFile.BinaryFile),
	}
}

func (r *MemoryFileRepository) SaveTypstFile(_ context.Context, f *domainFile.TypstFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.typstFiles[f.ID()] = f
	return nil
}

func (r *MemoryFileRepository) SaveBinaryFile(_ context.Context, f *domainFile.BinaryFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.binaryFiles[f.ID()] = f
	return nil
}

func (r *MemoryFileRepository) FindTypstFileByID(_ context.Context, id uuid.UUID) (*domainFile.TypstFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.typstFiles[id]
	if !ok {
		return nil, errors.New("typst file not found")
	}
	return f, nil
}

func (r *MemoryFileRepository) FindBinaryFileByID(_ context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.binaryFiles[id]
	if !ok {
		return nil, errors.New("binary file not found")
	}
	return f, nil
}

func (r *MemoryFileRepository) FindByProjectID(_ context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domainFile.File

	for _, f := range r.typstFiles {
		if f.ProjectID() == projectID {
			result = append(result, f)
		}
	}

	for _, f := range r.binaryFiles {
		if f.ProjectID() == projectID {
			result = append(result, f)
		}
	}

	return result, nil
}

func (r *MemoryFileRepository) DeleteFile(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, existsTypst := r.typstFiles[id]
	_, existsBinary := r.binaryFiles[id]

	if !existsTypst && !existsBinary {
		return errors.New("file not found")
	}

	delete(r.typstFiles, id)
	delete(r.binaryFiles, id)
	return nil
}
