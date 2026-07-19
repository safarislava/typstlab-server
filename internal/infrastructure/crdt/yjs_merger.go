package crdt

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

	"github.com/safarislava/typstlab-server/internal/domain/block"
)

type YjsMerger struct{}

func NewYjsMerger() *YjsMerger {
	return &YjsMerger{}
}

func (m *YjsMerger) MergeFile(state, delta []byte) (newState []byte, updatedBlocks []block.Block, err error) {
	doc := crdt.New()

	if len(state) > 0 {
		if err := doc.ApplyUpdate(state); err != nil {
			return nil, nil, fmt.Errorf("failed to apply current state update: %w", err)
		}
	}

	if len(delta) > 0 {
		if err := doc.ApplyUpdate(delta); err != nil {
			return nil, nil, fmt.Errorf("failed to apply delta update: %w", err)
		}
	}

	blocks := doc.GetArray("blocks").ToSlice()
	updatedBlocks = make([]block.Block, 0, len(blocks))

	for i, v := range blocks {
		b, err := parseBlockElement(v, doc)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse block element at index %d: %w", i, err)
		}
		updatedBlocks = append(updatedBlocks, b)
	}

	newState = doc.EncodeStateAsUpdate()
	return newState, updatedBlocks, nil
}

func parseBlockElement(v any, doc *crdt.Doc) (block.Block, error) {
	var idStr, name string

	switch v := v.(type) {
	case map[string]any:
		idStr, _ = v["id"].(string)
		name, _ = v["name"].(string)
	case *crdt.YMap:
		if idVal, ok := v.Get("id"); ok {
			idStr, _ = idVal.(string)
		}
		if nameVal, ok := v.Get("name"); ok {
			name, _ = nameVal.(string)
		}
	default:
		return block.Block{}, fmt.Errorf("invalid element type: %T", v)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to parse block uuid %q: %w", idStr, err)
	}

	text := doc.GetText("block:" + idStr)
	content := text.ToString()

	b, err := block.NewBlock(id, name, content)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to create block: %w", err)
	}

	return b, nil
}
