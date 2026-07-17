package block

import "github.com/google/uuid"

type Block struct {
	id        uuid.UUID
	name      string
	crdtState []byte
	content   string
}

func (b Block) ID() uuid.UUID {
	return b.id
}

func (b Block) Name() string {
	return b.name
}

func (b Block) CRDTState() []byte {
	if b.crdtState == nil {
		return nil
	}
	return append([]byte(nil), b.crdtState...)
}

func (b Block) Content() string {
	return b.content
}
