package serialization

import (
	"encoding/xml"
	"fmt"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

type xmlBlock struct {
	XMLName xml.Name `xml:"block"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
	Content string   `xml:",chardata"`
}

func serializeBlock(b block.Block) xmlBlock {
	return xmlBlock{
		ID:      b.ID().String(),
		Name:    b.Name(),
		Content: b.Content(),
	}
}

func deserializeBlock(xb *xmlBlock) (block.Block, error) {
	id, err := uuid.Parse(xb.ID)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to parse block uuid: %w", err)
	}

	b, err := block.NewBlock(id, xb.Name, xb.Content)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to create block from XML: %w", err)
	}

	return b, nil
}
