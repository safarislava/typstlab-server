package crdt

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

const (
	testIntroName    = "Introduction"
	testIntroContent = "Hello, CRDT world!"
	testSec1Name     = "Section 1"
	testSec1Content  = "Content of section 1"
	blockNameKey     = "name"
	testRenamedTyp   = "renamed.typ"
)

func TestYjsMerger_MergeFile_Initial(t *testing.T) {
	t.Parallel()

	merger := NewYjsMerger()

	doc := crdt.New()
	blockID1 := uuid.New()
	blockID2 := uuid.New()

	doc.Transact(func(txn *crdt.Transaction) {
		arr := txn.GetArray("blocks")

		m1 := map[string]any{
			"id":         blockID1.String(),
			blockNameKey: testIntroName,
		}
		t1 := txn.GetText("block:" + blockID1.String())
		t1.Insert(txn, 0, testIntroContent, nil)

		m2 := map[string]any{
			"id":         blockID2.String(),
			blockNameKey: testSec1Name,
		}
		t2 := txn.GetText("block:" + blockID2.String())
		t2.Insert(txn, 0, testSec1Content, nil)

		arr.Push(txn, []any{m1, m2})
	})

	delta := doc.EncodeStateAsUpdate()

	_, blocks, err := merger.MergeFile(nil, delta)
	if err != nil {
		t.Fatalf("failed to merge file: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].ID() != blockID1 || blocks[0].Name() != testIntroName || blocks[0].Content() != testIntroContent {
		t.Errorf("block 1 mismatch: %+v", blocks[0])
	}

	if blocks[1].ID() != blockID2 || blocks[1].Name() != testSec1Name || blocks[1].Content() != testSec1Content {
		t.Errorf("block 2 mismatch: %+v", blocks[1])
	}
}

func TestYjsMerger_MergeFile_UpdateAndSwap(t *testing.T) {
	t.Parallel()

	merger := NewYjsMerger()

	doc := crdt.New()
	blockID1 := uuid.New()
	blockID2 := uuid.New()

	doc.Transact(func(txn *crdt.Transaction) {
		arr := txn.GetArray("blocks")
		m1 := map[string]any{"id": blockID1.String(), blockNameKey: testIntroName}
		t1 := txn.GetText("block:" + blockID1.String())
		t1.Insert(txn, 0, testIntroContent, nil)

		m2 := map[string]any{"id": blockID2.String(), blockNameKey: testSec1Name}
		t2 := txn.GetText("block:" + blockID2.String())
		t2.Insert(txn, 0, testSec1Content, nil)

		arr.Push(txn, []any{m1, m2})
	})

	initialState := doc.EncodeStateAsUpdate()

	doc2 := crdt.New()
	if applyErr := doc2.ApplyUpdate(initialState); applyErr != nil {
		t.Fatalf("failed to apply state to doc2: %v", applyErr)
	}

	doc2.Transact(func(txn *crdt.Transaction) {
		arr := txn.GetArray("blocks")
		arr.Move(txn, 0, 1) // Swap order

		t1 := txn.GetText("block:" + blockID1.String())
		t1.Insert(txn, t1.Len(), " - Appended!", nil)
	})

	delta2 := doc2.EncodeStateAsUpdate()

	newState, blocks, err := merger.MergeFile(initialState, delta2)
	if err != nil {
		t.Fatalf("failed to merge second delta: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	// Blocks should be swapped
	if blocks[0].ID() != blockID2 || blocks[0].Content() != testSec1Content {
		t.Errorf("block 0 (swapped) mismatch: %+v", blocks[0])
	}

	if blocks[1].ID() != blockID1 || blocks[1].Content() != testIntroContent+" - Appended!" {
		t.Errorf("block 1 (swapped and modified) mismatch: %+v", blocks[1])
	}

	if bytes.Equal(initialState, newState) {
		t.Error("expected state updates to change the binary update state representation")
	}
}

func TestUserPayloadContent(t *testing.T) {
	t.Parallel()

	stateB64 := "AQv0iLN1AAcBBmJsb2NrcwEoAPSIs3UAAmlkAXckOTQzNTI4OTgtODlmNy00MjdhLTg2ZmEtM2E1MThhOGU3YzRiIQD0iLN1AARuYW1lASEA9IizdQAHY29udGVudAGh9IizdQMBofSIs3UEAaH0iLN1BQGh9IizdQYBofSIs3UHAaj0iLN1CAF3My8vIDEzLnR5cHhtbAog0LLRhNGLCgoK0LLRhNGL0LvRjNCy0LvQtNGE0YvQsgoK0LLQsqj0iLN1AgF3BtCy0YTRiwH0iLN1AQIH"
	data, err := base64.StdEncoding.DecodeString(stateB64)
	if err != nil {
		t.Fatalf("failed to decode b64: %v", err)
	}

	merger := NewYjsMerger()
	_, blocks, err := merger.MergeFile(data, nil)
	if err != nil {
		t.Fatalf("failed to merge file: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	expectedID := uuid.MustParse("94352898-89f7-427a-86fa-3a518a8e7c4b")
	if blocks[0].ID() != expectedID {
		t.Errorf("expected ID %s, got %s", expectedID, blocks[0].ID())
	}
	if blocks[0].Name() != "вфы" {
		t.Errorf("expected name %q, got %q", "вфы", blocks[0].Name())
	}
	if blocks[0].Content() == "" {
		t.Error("expected non-empty content, got empty string")
	}
}

func TestYjsMerger_SyncMetadata(t *testing.T) {
	t.Parallel()

	merger := NewYjsMerger()
	fileID := uuid.New()
	projectID := uuid.New()

	serverEntry, _ := domainEntry.NewEntry(fileID, "main.typ", domainFile.TypeTypst, false, time.Now())
	serverMeta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{serverEntry})

	// 1. Initial metadata sync with empty client vector
	clientDoc := crdt.New()
	sv := crdt.EncodeStateVectorV1(clientDoc)

	delta, meta, err := merger.SyncMetadata(projectID, serverMeta, nil, sv)
	if err != nil {
		t.Fatalf("failed to sync metadata: %v", err)
	}

	if len(delta) == 0 {
		t.Error("expected non-empty metadata delta")
	}
	gotEntry, exists := meta.Get(fileID)
	if !exists || gotEntry.Name() != "main.typ" {
		t.Errorf("unexpected active entries: %+v", meta.Entries())
	}

	// 2. Client applies delta and modifies metadata
	if applyErr := clientDoc.ApplyUpdate(delta); applyErr != nil {
		t.Fatalf("failed to apply update to clientDoc: %v", applyErr)
	}

	clientDoc.Transact(func(txn *crdt.Transaction) {
		filesMap := txn.GetMap("files")
		fileEntry := map[string]any{
			"id":         fileID.String(),
			"name":       testRenamedTyp,
			"type":       string(domainFile.TypeTypst),
			"is_deleted": false,
		}
		filesMap.Set(txn, fileID.String(), fileEntry)
	})
	clientDelta := clientDoc.EncodeStateAsUpdate()

	_, updatedMeta, err := merger.SyncMetadata(projectID, serverMeta, clientDelta, nil)
	if err != nil {
		t.Fatalf("failed to sync updated metadata: %v", err)
	}

	updatedEntry, exists := updatedMeta.Get(fileID)
	if !exists || updatedEntry.Name() != testRenamedTyp {
		t.Errorf("expected renamed file name %q, got %+v", testRenamedTyp, updatedEntry)
	}
}

func TestYjsMerger_ComputeDelta(t *testing.T) {
	t.Parallel()

	merger := NewYjsMerger()

	serverDoc := crdt.New()
	serverDoc.Transact(func(txn *crdt.Transaction) {
		t1 := txn.GetText("block:test")
		t1.Insert(txn, 0, "Hello World", nil)
	})
	serverState := serverDoc.EncodeStateAsUpdate()

	clientDoc := crdt.New()
	clientSV := crdt.EncodeStateVectorV1(clientDoc)

	delta, err := merger.ComputeDelta(serverState, clientSV)
	if err != nil {
		t.Fatalf("failed to compute delta: %v", err)
	}

	if len(delta) == 0 {
		t.Error("expected non-empty delta update")
	}

	if applyErr := clientDoc.ApplyUpdate(delta); applyErr != nil {
		t.Fatalf("failed to apply delta to clientDoc: %v", applyErr)
	}

	txt := clientDoc.GetText("block:test").ToString()
	if txt != "Hello World" {
		t.Errorf("expected text %q, got %q", "Hello World", txt)
	}
}

func TestYjsMerger_SyncMetadata_WithYMap(t *testing.T) {
	t.Parallel()

	merger := NewYjsMerger()
	fileID := uuid.New()
	projectID := uuid.New()

	serverEntry, _ := domainEntry.NewEntry(fileID, "main.typ", domainFile.TypeTypst, false, time.Now())
	serverMeta, _ := domainMeta.NewMetadata(projectID, []*domainEntry.Entry{serverEntry})

	clientDoc := crdt.New()
	clientDoc.Transact(func(txn *crdt.Transaction) {
		filesMap := txn.GetMap("files")
		nested := crdt.NewMapPrelim()
		nested.Set(txn, "id", fileID.String())
		nested.Set(txn, "name", "from_ymap.typ")
		nested.Set(txn, "type", string(domainFile.TypeTypst))
		nested.Set(txn, "is_deleted", true)
		filesMap.Set(txn, fileID.String(), nested)
	})

	clientDelta := clientDoc.EncodeStateAsUpdate()

	_, updatedMeta, err := merger.SyncMetadata(projectID, serverMeta, clientDelta, nil)
	if err != nil {
		t.Fatalf("failed to sync updated metadata with YMap: %v", err)
	}

	updatedEntry, exists := updatedMeta.Get(fileID)
	if !exists {
		t.Fatal("expected entry to exist in metadata")
	}
	if updatedEntry.Name() != "from_ymap.typ" || !updatedEntry.IsDeleted() {
		t.Errorf("unexpected entry values: %+v", updatedEntry)
	}
}

func TestYjsMerger_MergeFile_WithYMapBlocks(t *testing.T) {
	t.Parallel()

	merger := NewYjsMerger()
	doc := crdt.New()
	blockID := uuid.New()

	doc.Transact(func(txn *crdt.Transaction) {
		arr := txn.GetArray("blocks")
		nested := crdt.NewMapPrelim()
		nested.Set(txn, "id", blockID.String())
		nested.Set(txn, "name", "Header Block")
		nested.Set(txn, "content", "Direct content")
		arr.PushType(txn, nested)
	})

	delta := doc.EncodeStateAsUpdate()

	_, blocks, err := merger.MergeFile(nil, delta)
	if err != nil {
		t.Fatalf("failed to merge file with YMap block: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].ID() != blockID || blocks[0].Name() != "Header Block" || blocks[0].Content() != "Direct content" {
		t.Errorf("unexpected block values: %+v", blocks[0])
	}
}
