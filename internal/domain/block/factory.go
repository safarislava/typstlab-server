package block

import "github.com/google/uuid"

func NewBlock(id uuid.UUID, name string, state []byte, content string) (Block, error) {
	if id == uuid.Nil {
		return Block{}, ErrEmptyBlockID
	}
	if state == nil {
		return Block{}, ErrEmptyBlockState
	}
	return Block{
		id:      id,
		name:    name,
		state:   append([]byte(nil), state...),
		content: content,
	}, nil
}
