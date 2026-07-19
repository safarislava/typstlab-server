package serialization

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

type xmlBlock struct {
	XMLName xml.Name `xml:"block"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
	State   string   `xml:"state,attr"`
	Content string   `xml:",chardata"`
}

func serializeBlock(b block.Block) xmlBlock {
	return xmlBlock{
		ID:      b.ID().String(),
		Name:    b.Name(),
		State:   base64.StdEncoding.EncodeToString(b.State()),
		Content: b.Content(),
	}
}

func deserializeBlock(xb *xmlBlock) (block.Block, error) {
	id, err := uuid.Parse(xb.ID)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to parse block uuid: %w", err)
	}

	state, err := base64.StdEncoding.DecodeString(xb.State)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to decode block state from base64: %w", err)
	}

	if len(state) == 0 {
		state = nil
	}

	b, err := block.NewBlock(id, xb.Name, state, xb.Content)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to create block from XML: %w", err)
	}

	return b, nil
}
