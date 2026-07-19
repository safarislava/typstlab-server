package crdt

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"
)

const (
	testIntroName    = "Introduction"
	testIntroContent = "Hello, CRDT world!"
	testSec1Name     = "Section 1"
	testSec1Content  = "Content of section 1"
	blockNameKey     = "name"
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
