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
	var idStr, name, content string

	switch v := v.(type) {
	case map[string]any:
		idStr = extractString(v["id"])
		name = extractString(v["name"])
		content = extractString(v["content"])
	case *crdt.YMap:
		if idVal, ok := v.Get("id"); ok {
			idStr = extractString(idVal)
		}
		if nameVal, ok := v.Get("name"); ok {
			name = extractString(nameVal)
		}
		if contentVal, ok := v.Get("content"); ok {
			content = extractString(contentVal)
		}
	default:
		return block.Block{}, fmt.Errorf("invalid element type: %T", v)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to parse block uuid %q: %w", idStr, err)
	}

	if content == "" {
		text := doc.GetText("block:" + idStr)
		content = text.ToString()
	}

	b, err := block.NewBlock(id, name, content)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to create block: %w", err)
	}

	return b, nil
}

func extractString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case *crdt.YText:
		if v != nil {
			return v.ToString()
		}
	case fmt.Stringer:
		if v != nil {
			return v.String()
		}
	}
	return ""
}
