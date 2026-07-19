package file

import (
	"github.com/safarislava/typstlab-server/internal/domain/block"
)

type Merger interface {
	MergeFile(state, delta []byte) (newState []byte, updatedBlocks []block.Block, err error)
}
