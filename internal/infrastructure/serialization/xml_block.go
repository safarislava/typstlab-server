package serialization

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

type xmlBlock struct {
	XMLName   xml.Name `xml:"block"`
	ID        string   `xml:"id,attr"`
	Name      string   `xml:"name,attr"`
	CRDTState string   `xml:"crdt_state,attr"`
	Content   string   `xml:",chardata"`
}

func serializeBlock(b block.Block) xmlBlock {
	return xmlBlock{
		ID:        b.ID().String(),
		Name:      b.Name(),
		CRDTState: base64.StdEncoding.EncodeToString(b.CRDTState()),
		Content:   b.Content(),
	}
}

func deserializeBlock(xb *xmlBlock) (block.Block, error) {
	id, err := uuid.Parse(xb.ID)
	if err != nil {
		return block.Block{}, fmt.Errorf("could not parse block ID: %w", err)
	}

	crdtState, err := base64.StdEncoding.DecodeString(xb.CRDTState)
	if err != nil {
		return block.Block{}, fmt.Errorf("could not decode CRDT state: %w", err)
	}

	if len(crdtState) == 0 {
		crdtState = nil
	}

	b, err := block.NewBlock(id, xb.Name, crdtState, xb.Content)
	if err != nil {
		return block.Block{}, fmt.Errorf("could not initialize block: %w", err)
	}

	return b, nil
}
