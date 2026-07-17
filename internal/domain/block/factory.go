package block

import "github.com/google/uuid"

func NewBlock(id uuid.UUID, name string, crdtState []byte, content string) (Block, error) {
	if id == uuid.Nil {
		return Block{}, ErrEmptyBlockID
	}
	if crdtState == nil {
		return Block{}, ErrEmptyBlockCrdt
	}
	return Block{
		id:        id,
		name:      name,
		crdtState: append([]byte(nil), crdtState...),
		content:   content,
	}, nil
}
