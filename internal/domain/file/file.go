package file

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

type Type string

const (
	TypeBinary Type = "binary"
	TypeTypst  Type = "typst"
)

type File interface {
	ID() uuid.UUID
	ProjectID() uuid.UUID
	Name() string
	Type() Type
	UpdatedAt() time.Time
}

type BinaryFile struct {
	id        uuid.UUID
	projectID uuid.UUID
	name      string
	content   []byte
	updatedAt time.Time
}

func (f *BinaryFile) ID() uuid.UUID {
	return f.id
}

func (f *BinaryFile) ProjectID() uuid.UUID {
	return f.projectID
}

func (f *BinaryFile) Name() string {
	return f.name
}

func (f *BinaryFile) Type() Type {
	return TypeBinary
}

func (f *BinaryFile) UpdatedAt() time.Time {
	return f.updatedAt
}

func (f *BinaryFile) Content() []byte {
	if f.content == nil {
		return nil
	}
	return append([]byte(nil), f.content...)
}

type TypstFile struct {
	id        uuid.UUID
	projectID uuid.UUID
	name      string
	blocks    []block.Block
	updatedAt time.Time
}

func (f *TypstFile) ID() uuid.UUID {
	return f.id
}

func (f *TypstFile) ProjectID() uuid.UUID {
	return f.projectID
}

func (f *TypstFile) Name() string {
	return f.name
}

func (f *TypstFile) Type() Type {
	return TypeTypst
}

func (f *TypstFile) UpdatedAt() time.Time {
	return f.updatedAt
}

func (f *TypstFile) Blocks() []block.Block {
	if f.blocks == nil {
		return nil
	}
	return append([]block.Block(nil), f.blocks...)
}

func (f *TypstFile) FindBlockByID(blockID uuid.UUID) (block.Block, error) {
	for _, b := range f.blocks {
		if b.ID() == blockID {
			return b, nil
		}
	}
	return block.Block{}, ErrBlockNotFound
}

func (f *TypstFile) UpdateBlock(blockID uuid.UUID, state []byte, content string) error {
	for i, b := range f.blocks {
		if b.ID() != blockID {
			continue
		}
		newBlock, err := block.NewBlock(blockID, b.Name(), state, content)
		if err != nil {
			return fmt.Errorf("failed to create block: %w", err)
		}
		f.blocks[i] = newBlock
		f.updatedAt = time.Now()
		return nil
	}
	return ErrBlockNotFound
}
