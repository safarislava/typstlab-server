package serialization

import (
	"encoding/base64"
	"encoding/xml"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

// xmlBlock is the XML representation of a block with base64-encoded state.
type xmlBlock struct {
	XMLName xml.Name `xml:"block"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
	State   string   `xml:"state,attr"`
	Content string   `xml:",chardata"`
}

// serializeBlock converts a domain Block to its XML representation.
func serializeBlock(b block.Block) xmlBlock {
	return xmlBlock{
		ID:      b.ID().String(),
		Name:    b.Name(),
		State:   base64.StdEncoding.EncodeToString(b.State()),
		Content: b.Content(),
	}
}

// deserializeBlock converts an XML block representation to a domain Block.
// Validation is performed through the domain factory.
func deserializeBlock(xb *xmlBlock) (block.Block, error) {
	id, err := uuid.Parse(xb.ID)
	if err != nil {
		return block.Block{}, err
	}

	state, err := base64.StdEncoding.DecodeString(xb.State)
	if err != nil {
		return block.Block{}, err
	}

	if len(state) == 0 {
		state = nil
	}

	return block.NewBlock(id, xb.Name, state, xb.Content)
}
