package block

import "github.com/google/uuid"

func NewBlock(id uuid.UUID, name, content string) (Block, error) {
	if id == uuid.Nil {
		return Block{}, ErrEmptyBlockID
	}
	return Block{
		id:      id,
		name:    name,
		content: content,
	}, nil
}
