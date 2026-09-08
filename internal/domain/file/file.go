package file

import (
	"time"

	"github.com/google/uuid"
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

func (f *BinaryFile) Rename(name string) error {
	if name == "" {
		return ErrEmptyFileName
	}
	f.name = name
	f.updatedAt = time.Now()
	return nil
}

type TypstFile struct {
	id        uuid.UUID
	projectID uuid.UUID
	name      string
	state     []byte
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

func (f *TypstFile) State() []byte {
	if f.state == nil {
		return nil
	}
	return append([]byte(nil), f.state...)
}

func (f *TypstFile) UpdateState(state []byte) error {
	if state == nil {
		return ErrNilState
	}
	f.state = append([]byte(nil), state...)
	f.updatedAt = time.Now()
	return nil
}

func (f *TypstFile) Rename(name string) error {
	if name == "" {
		return ErrEmptyFileName
	}
	f.name = name
	f.updatedAt = time.Now()
	return nil
}
