package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type MemoryFileRepository struct {
	mu          sync.RWMutex
	typstFiles  map[uuid.UUID]*domainFile.TypstFile
	binaryFiles map[uuid.UUID]*domainFile.BinaryFile
	tombstones  map[uuid.UUID]bool
}

func NewMemoryFileRepository() *MemoryFileRepository {
	return &MemoryFileRepository{
		typstFiles:  make(map[uuid.UUID]*domainFile.TypstFile),
		binaryFiles: make(map[uuid.UUID]*domainFile.BinaryFile),
		tombstones:  make(map[uuid.UUID]bool),
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
		return nil, domainFile.ErrTypstFileNotFound
	}
	return f, nil
}

func (r *MemoryFileRepository) FindBinaryFileByID(_ context.Context, id uuid.UUID) (*domainFile.BinaryFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.binaryFiles[id]
	if !ok {
		return nil, domainFile.ErrBinaryFileNotFound
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
		return domainFile.ErrFileNotFound
	}

	delete(r.typstFiles, id)
	delete(r.binaryFiles, id)
	r.tombstones[id] = true
	return nil
}

func (r *MemoryFileRepository) IsDeleted(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tombstones[id], nil
}

func (r *MemoryFileRepository) FindEntriesByProjectID(_ context.Context, projectID uuid.UUID) ([]*domainEntry.Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domainEntry.Entry

	var err error
	result, err = appendEntriesForTypst(result, r.typstFiles, projectID)
	if err != nil {
		return nil, err
	}

	result, err = appendEntriesForBinary(result, r.binaryFiles, projectID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func appendEntriesForTypst(
	result []*domainEntry.Entry,
	files map[uuid.UUID]*domainFile.TypstFile,
	projectID uuid.UUID,
) ([]*domainEntry.Entry, error) {
	for _, f := range files {
		if f.ProjectID() == projectID {
			entry, err := domainEntry.NewEntry(f.ID(), f.Name(), f.Type(), false, f.UpdatedAt())
			if err != nil {
				return nil, fmt.Errorf("failed to create entry for typst file %s: %w", f.ID(), err)
			}
			result = append(result, entry)
		}
	}
	return result, nil
}

func appendEntriesForBinary(
	result []*domainEntry.Entry,
	files map[uuid.UUID]*domainFile.BinaryFile,
	projectID uuid.UUID,
) ([]*domainEntry.Entry, error) {
	for _, f := range files {
		if f.ProjectID() == projectID {
			entry, err := domainEntry.NewEntry(f.ID(), f.Name(), f.Type(), false, f.UpdatedAt())
			if err != nil {
				return nil, fmt.Errorf("failed to create entry for binary file %s: %w", f.ID(), err)
			}
			result = append(result, entry)
		}
	}
	return result, nil
}
