package block

import "github.com/google/uuid"

type Block struct {
	id      uuid.UUID
	name    string
	content string
}

func (b Block) ID() uuid.UUID {
	return b.id
}

func (b Block) Name() string {
	return b.name
}

func (b Block) Content() string {
	return b.content
}
