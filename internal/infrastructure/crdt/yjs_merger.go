package crdt

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

const (
	keyID        = "id"
	keyName      = "name"
	keyType      = "type"
	keyIsDeleted = "is_deleted"
	keyFiles     = "files"
	keyBlocks    = "blocks"
)

type YjsMerger struct{}

func NewYjsMerger() *YjsMerger {
	return &YjsMerger{}
}

func (m *YjsMerger) MergeFile(state, delta []byte) ([]byte, []block.Block, error) {
	doc := crdt.New()

	if err := applyUpdates(doc, state, delta); err != nil {
		return nil, nil, err
	}

	updatedBlocks, err := extractBlocks(doc)
	if err != nil {
		return nil, nil, err
	}

	return doc.EncodeStateAsUpdate(), updatedBlocks, nil
}

func (m *YjsMerger) SyncMetadata(
	projectID uuid.UUID,
	currentMeta *domainMeta.Metadata,
	clientDelta []byte,
	clientStateVector []byte,
) (metadataDelta []byte, meta *domainMeta.Metadata, err error) {
	doc := crdt.New()

	if len(clientDelta) > 0 {
		if applyErr := doc.ApplyUpdate(clientDelta); applyErr != nil {
			return nil, nil, fmt.Errorf("failed to apply client metadata delta: %w", applyErr)
		}
	}

	populateMetadataDoc(doc, currentMeta)

	entries, err := extractMetadataEntries(doc)
	if err != nil {
		return nil, nil, err
	}

	meta, err = domainMeta.NewMetadata(projectID, entries)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create project metadata: %w", err)
	}

	metadataDelta, err = computeMetadataDelta(doc, clientStateVector, len(clientDelta) > 0)
	if err != nil {
		return nil, nil, err
	}

	return metadataDelta, meta, nil
}

func (m *YjsMerger) ComputeDelta(serverState, clientStateVector []byte) ([]byte, error) {
	doc := crdt.New()
	if len(serverState) > 0 {
		if err := doc.ApplyUpdate(serverState); err != nil {
			return nil, fmt.Errorf("failed to apply server state update: %w", err)
		}
	}

	stateVector, err := crdt.DecodeStateVectorV1(clientStateVector)
	if err != nil {
		return nil, fmt.Errorf("failed to decode client state vector: %w", err)
	}

	return crdt.EncodeStateAsUpdateV1(doc, stateVector), nil
}

func populateMetadataDoc(doc *crdt.Doc, currentMeta *domainMeta.Metadata) {
	if currentMeta == nil {
		return
	}
	filesMap := doc.GetMap(keyFiles)
	existingKeys := make(map[string]bool)
	if filesMap != nil {
		for _, k := range filesMap.Keys() {
			existingKeys[k] = true
		}
	}

	doc.Transact(func(txn *crdt.Transaction) {
		m := txn.GetMap(keyFiles)
		for _, entry := range currentMeta.Entries() {
			if existingKeys[entry.ID().String()] {
				continue
			}
			fileEntry := map[string]any{
				keyID:        entry.ID().String(),
				keyName:      entry.Name(),
				keyType:      string(entry.Type()),
				keyIsDeleted: entry.IsDeleted(),
			}
			m.Set(txn, entry.ID().String(), fileEntry)
		}
	})
}

func extractMetadataEntries(doc *crdt.Doc) ([]*domainEntry.Entry, error) {
	filesMap := doc.GetMap(keyFiles)
	if filesMap == nil {
		return nil, nil
	}

	keys := filesMap.Keys()
	entries := make([]*domainEntry.Entry, 0, len(keys))

	for _, k := range keys {
		val, ok := filesMap.Get(k)
		if !ok {
			continue
		}
		entry, err := parseMetadataEntry(k, val)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func parseMetadataEntry(key string, val any) (*domainEntry.Entry, error) {
	var idStr, name, typeStr string
	var isDeleted bool

	switch v := val.(type) {
	case map[string]any:
		idStr, name, typeStr, isDeleted = parseMetadataFromMap(v)
	case *crdt.YMap:
		idStr, name, typeStr, isDeleted = parseMetadataFromYMap(v)
	default:
		return nil, fmt.Errorf("invalid metadata entry type: %T", val)
	}

	if idStr == "" {
		idStr = key
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata file uuid %q: %w", idStr, err)
	}

	entry, err := domainEntry.NewEntry(id, name, domainFile.Type(typeStr), isDeleted, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create entry: %w", err)
	}

	return entry, nil
}

func parseMetadataFromMap(v map[string]any) (id, name, typeStr string, isDeleted bool) {
	id = extractString(v[keyID])
	name = extractString(v[keyName])
	typeStr = extractString(v[keyType])
	if del, ok := v[keyIsDeleted].(bool); ok {
		isDeleted = del
	}
	return id, name, typeStr, isDeleted
}

func parseMetadataFromYMap(v *crdt.YMap) (id, name, typeStr string, isDeleted bool) {
	if val, ok := v.Get(keyID); ok {
		id = extractString(val)
	}
	if val, ok := v.Get(keyName); ok {
		name = extractString(val)
	}
	if val, ok := v.Get(keyType); ok {
		typeStr = extractString(val)
	}
	if val, ok := v.Get(keyIsDeleted); ok {
		if del, ok := val.(bool); ok {
			isDeleted = del
		}
	}
	return id, name, typeStr, isDeleted
}

func computeMetadataDelta(doc *crdt.Doc, stateVectorBytes []byte, hasClientDelta bool) ([]byte, error) {
	if len(stateVectorBytes) > 0 {
		sv, err := crdt.DecodeStateVectorV1(stateVectorBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decode client metadata state vector: %w", err)
		}
		return crdt.EncodeStateAsUpdateV1(doc, sv), nil
	}

	if hasClientDelta {
		return doc.EncodeStateAsUpdate(), nil
	}

	return nil, nil
}

func applyUpdates(doc *crdt.Doc, state, delta []byte) error {
	if len(state) > 0 {
		if err := doc.ApplyUpdate(state); err != nil {
			return fmt.Errorf("failed to apply current state update: %w", err)
		}
	}
	if len(delta) > 0 {
		if err := doc.ApplyUpdate(delta); err != nil {
			return fmt.Errorf("failed to apply delta update: %w", err)
		}
	}
	return nil
}

func extractBlocks(doc *crdt.Doc) ([]block.Block, error) {
	blocks := doc.GetArray(keyBlocks).ToSlice()
	updatedBlocks := make([]block.Block, 0, len(blocks))
	seenIDs := make(map[uuid.UUID]bool)

	for i, v := range blocks {
		b, err := parseBlockElement(v, doc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse block element at index %d: %w", i, err)
		}
		if !seenIDs[b.ID()] {
			seenIDs[b.ID()] = true
			updatedBlocks = append(updatedBlocks, b)
		}
	}

	return updatedBlocks, nil
}

func parseBlockElement(v any, doc *crdt.Doc) (block.Block, error) {
	var idStr, name, content string

	switch v := v.(type) {
	case map[string]any:
		idStr = extractString(v[keyID])
		name = extractString(v[keyName])
		content = extractString(v["content"])
	case *crdt.YMap:
		if idVal, ok := v.Get(keyID); ok {
			idStr = extractString(idVal)
		}
		if nameVal, ok := v.Get(keyName); ok {
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
