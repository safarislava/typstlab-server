package crdt

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

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
)

// YjsMerger provides CRDT document merging and delta computation.
type YjsMerger struct{}

// NewYjsMerger creates a new YjsMerger instance.
func NewYjsMerger() *YjsMerger {
	return &YjsMerger{}
}

// MergeFile merges an existing file state update with an incoming delta update.
func (m *YjsMerger) MergeFile(state, delta []byte) ([]byte, error) {
	doc, err := newDocWithUpdates(state, delta)
	if err != nil {
		return nil, fmt.Errorf("merge file updates: %w", err)
	}
	return doc.EncodeStateAsUpdate(), nil
}

// ComputeDelta computes the delta between server state and a client state vector.
func (m *YjsMerger) ComputeDelta(serverState, clientStateVector []byte) ([]byte, error) {
	doc, err := newDocWithUpdates(serverState)
	if err != nil {
		return nil, fmt.Errorf("apply server state: %w", err)
	}

	stateVector, err := decodeStateVector(clientStateVector)
	if err != nil {
		return nil, fmt.Errorf("decode client state vector: %w", err)
	}

	return crdt.EncodeStateAsUpdateV1(doc, stateVector), nil
}

// SyncMetadata merges client metadata delta with current metadata and returns delta & updated aggregate.
func (m *YjsMerger) SyncMetadata(
	projectID uuid.UUID,
	currentMeta *domainMeta.Metadata,
	clientDelta []byte,
	clientStateVector []byte,
) (metadataDelta []byte, meta *domainMeta.Metadata, err error) {
	doc, err := newDocWithUpdates(clientDelta)
	if err != nil {
		return nil, nil, fmt.Errorf("apply client metadata delta: %w", err)
	}

	mergeServerMetadata(doc, currentMeta)

	updatedMeta, err := extractProjectMetadata(doc, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("extract metadata: %w", err)
	}

	delta, err := computeMetadataDelta(doc, clientStateVector, len(clientDelta) > 0)
	if err != nil {
		return nil, nil, fmt.Errorf("compute metadata delta: %w", err)
	}

	return delta, updatedMeta, nil
}

func newDocWithUpdates(updates ...[]byte) (*crdt.Doc, error) {
	doc := crdt.New()
	for _, u := range updates {
		if len(u) == 0 {
			continue
		}
		if err := doc.ApplyUpdate(u); err != nil {
			return nil, fmt.Errorf("apply update: %w", err)
		}
	}
	return doc, nil
}

func mergeServerMetadata(doc *crdt.Doc, currentMeta *domainMeta.Metadata) {
	if currentMeta == nil {
		return
	}

	existingKeys := getMapKeys(doc, keyFiles)
	doc.Transact(func(txn *crdt.Transaction) {
		filesMap := txn.GetMap(keyFiles)
		for _, entry := range currentMeta.Entries() {
			setEntryIfMissing(txn, filesMap, existingKeys, entry)
		}
	})
}

func setEntryIfMissing(txn *crdt.Transaction, filesMap *crdt.YMap, existingKeys map[string]bool, entry *domainEntry.Entry) {
	idStr := entry.ID().String()
	if existingKeys[idStr] {
		return
	}
	filesMap.Set(txn, idStr, map[string]any{
		keyID:        idStr,
		keyName:      entry.Name(),
		keyType:      string(entry.Type()),
		keyIsDeleted: entry.IsDeleted(),
	})
}

func getMapKeys(doc *crdt.Doc, mapKey string) map[string]bool {
	filesMap := doc.GetMap(mapKey)
	if filesMap == nil {
		return nil
	}

	keys := make(map[string]bool, len(filesMap.Keys()))
	for _, k := range filesMap.Keys() {
		keys[k] = true
	}
	return keys
}

func extractProjectMetadata(doc *crdt.Doc, projectID uuid.UUID) (*domainMeta.Metadata, error) {
	entries, err := extractEntries(doc)
	if err != nil {
		return nil, err
	}
	meta, err := domainMeta.NewMetadata(projectID, entries)
	if err != nil {
		return nil, fmt.Errorf("construct metadata aggregate: %w", err)
	}
	return meta, nil
}

func extractEntries(doc *crdt.Doc) ([]*domainEntry.Entry, error) {
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
		entry, err := parseEntry(k, val)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseEntry(fallbackID string, val any) (*domainEntry.Entry, error) {
	id, err := extractEntryID(val, fallbackID)
	if err != nil {
		return nil, err
	}

	name := getString(val, keyName)
	typeStr := getString(val, keyType)
	isDeleted := getBool(val, keyIsDeleted)

	entry, err := domainEntry.NewEntry(id, name, domainFile.Type(typeStr), isDeleted, time.Now())
	if err != nil {
		return nil, fmt.Errorf("construct entry: %w", err)
	}

	return entry, nil
}

func extractEntryID(val any, fallback string) (uuid.UUID, error) {
	idStr := getString(val, keyID)
	if idStr == "" {
		idStr = fallback
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid metadata file uuid %q: %w", idStr, err)
	}
	return id, nil
}

func computeMetadataDelta(doc *crdt.Doc, stateVectorBytes []byte, hasClientDelta bool) ([]byte, error) {
	if len(stateVectorBytes) > 0 {
		return encodeDocWithStateVector(doc, stateVectorBytes)
	}

	if hasClientDelta {
		return doc.EncodeStateAsUpdate(), nil
	}

	return nil, nil
}

func encodeDocWithStateVector(doc *crdt.Doc, stateVectorBytes []byte) ([]byte, error) {
	sv, err := decodeStateVector(stateVectorBytes)
	if err != nil {
		return nil, err
	}
	return crdt.EncodeStateAsUpdateV1(doc, sv), nil
}

func decodeStateVector(data []byte) (crdt.StateVector, error) {
	sv, err := crdt.DecodeStateVectorV1(data)
	if err != nil {
		return nil, fmt.Errorf("decode state vector: %w", err)
	}
	return sv, nil
}

func getValue(item any, key string) (any, bool) {
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

func getString(item any, key string) string {
	val, ok := getValue(item, key)
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

func getBool(item any, key string) bool {
	val, ok := getValue(item, key)
	if !ok {
		return false
	}
	b, _ := val.(bool)
	return b
}
