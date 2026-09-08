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
	keyContent   = "content"
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
	if applyErr := applyUpdates(doc, clientDelta); applyErr != nil {
		return nil, nil, fmt.Errorf("failed to apply client metadata delta: %w", applyErr)
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
	if err := applyUpdates(doc, serverState); err != nil {
		return nil, fmt.Errorf("failed to apply server state update: %w", err)
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
			idStr := entry.ID().String()
			if existingKeys[idStr] {
				continue
			}
			m.Set(txn, idStr, map[string]any{
				keyID:        idStr,
				keyName:      entry.Name(),
				keyType:      string(entry.Type()),
				keyIsDeleted: entry.IsDeleted(),
			})
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
	idStr := getStringField(val, keyID)
	if idStr == "" {
		idStr = key
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata file uuid %q: %w", idStr, err)
	}

	name := getStringField(val, keyName)
	typeStr := getStringField(val, keyType)
	isDeleted := getBoolField(val, keyIsDeleted)

	entry, err := domainEntry.NewEntry(id, name, domainFile.Type(typeStr), isDeleted, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create entry: %w", err)
	}

	return entry, nil
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
	idStr := getStringField(v, keyID)
	if idStr == "" {
		return block.Block{}, fmt.Errorf("invalid element type: %T", v)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to parse block uuid %q: %w", idStr, err)
	}

	content := getStringField(v, keyContent)
	if content == "" {
		content = doc.GetText("block:" + idStr).ToString()
	}

	b, err := block.NewBlock(id, getStringField(v, keyName), content)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to create block: %w", err)
	}

	return b, nil
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

func applyUpdates(doc *crdt.Doc, updates ...[]byte) error {
	for _, u := range updates {
		if len(u) > 0 {
			if err := doc.ApplyUpdate(u); err != nil {
				return fmt.Errorf("failed to apply update: %w", err)
			}
		}
	}
	return nil
}

func getField(item any, key string) (any, bool) {
	switch v := item.(type) {
	case map[string]any:
		val, ok := v[key]
		return val, ok
	case *crdt.YMap:
		return v.Get(key)
	default:
		return nil, false
	}
}

func getStringField(item any, key string) string {
	val, ok := getField(item, key)
	if !ok {
		return ""
	}
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

func getBoolField(item any, key string) bool {
	val, ok := getField(item, key)
	if !ok {
		return false
	}
	b, _ := val.(bool)
	return b
}
