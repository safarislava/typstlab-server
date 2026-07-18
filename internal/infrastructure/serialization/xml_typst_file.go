package serialization

import (
	"encoding/xml"
	"fmt"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	"github.com/safarislava/typstlab-server/internal/domain/file"
)

type xmlTypstFile struct {
	XMLName xml.Name   `xml:"file"`
	Blocks  []xmlBlock `xml:"block"`
}

func SerializeTypstFile(f *file.TypstFile) ([]byte, error) {
	blocks := f.Blocks()
	typstFile := xmlTypstFile{
		Blocks: make([]xmlBlock, len(blocks)),
	}

	for i, b := range blocks {
		typstFile.Blocks[i] = serializeBlock(b)
	}

	serialized, err := xml.MarshalIndent(typstFile, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize typst file: %w", err)
	}

	return serialized, nil
}

func DeserializeTypstFile(data []byte) ([]block.Block, error) {
	var doc xmlTypstFile
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize typst file: %w", err)
	}

	blocks := make([]block.Block, 0, len(doc.Blocks))
	for _, xb := range doc.Blocks {
		b, err := deserializeBlock(&xb)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}

	return blocks, nil
}
