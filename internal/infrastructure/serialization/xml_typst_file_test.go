package serialization

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	"github.com/safarislava/typstlab-server/internal/domain/file"
)

var testBlockID2 = uuid.MustParse("20000000-0000-0000-0000-000000000002")

func newTestTypstFile(t *testing.T, state []byte, blocks []block.Block) *file.TypstFile {
	t.Helper()
	f, err := file.NewTypstFile(uuid.New(), uuid.New(), "document.typ", state, blocks, time.Now())
	if err != nil {
		t.Fatalf("Failed to create TypstFile: %v", err)
	}
	return f
}

func TestSerializeTypstFile(t *testing.T) {
	t.Parallel()

	b1, err := block.NewBlock(testBlockID1, "Введение", "= Введение\nТекст введения")
	if err != nil {
		t.Fatalf("Failed to create block 1: %v", err)
	}
	b2, err := block.NewBlock(testBlockID2, "Глава 1", "= Глава 1\nТекст главы")
	if err != nil {
		t.Fatalf("Failed to create block 2: %v", err)
	}
	globalState := []byte("global-crdt-state")

	f := newTestTypstFile(t, globalState, []block.Block{b1, b2})

	data, err := SerializeTypstFile(f)
	if err != nil {
		t.Fatalf("SerializeTypstFile failed: %v", err)
	}

	xmlStr := string(data)

	if !strings.Contains(xmlStr, "<file") {
		t.Errorf("Expected <file> root element, got:\n%s", xmlStr)
	}
	expectedStateB64 := base64.StdEncoding.EncodeToString(globalState)
	if !strings.Contains(xmlStr, `state="`+expectedStateB64+`"`) {
		t.Errorf("Expected global state in file element, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `id="10000000-0000-0000-0000-000000000001"`) {
		t.Errorf("Expected first block id, got:\n%s", xmlStr)
	}
}

func TestDeserializeTypstFile(t *testing.T) {
	t.Parallel()

	globalState := []byte("global-crdt-state")
	stateB64 := base64.StdEncoding.EncodeToString(globalState)

	xmlData := `<file state="` + stateB64 + `">
    <block id="10000000-0000-0000-0000-000000000001" name="Введение">= Введение
Текст введения</block>
    <block id="20000000-0000-0000-0000-000000000002" name="Глава 1">= Глава 1
Текст главы</block>
</file>`

	state, blocks, err := DeserializeTypstFile([]byte(xmlData))
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}

	if !bytes.Equal(state, globalState) {
		t.Errorf("expected state %s, got %s", globalState, state)
	}

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].ID() != testBlockID1 {
		t.Errorf("Block 0: expected ID %s, got %s", testBlockID1, blocks[0].ID())
	}
}

func TestSerializeDeserializeTypstFile_Roundtrip(t *testing.T) {
	t.Parallel()

	globalState := []byte{0x01, 0x02, 0x03, 0x04}
	originalBlock, err := block.NewBlock(testBlockID1, "Test Block", "= Test\nContent here")
	if err != nil {
		t.Fatalf("Failed to create block: %v", err)
	}

	f := newTestTypstFile(t, globalState, []block.Block{originalBlock})

	data, err := SerializeTypstFile(f)
	if err != nil {
		t.Fatalf("SerializeTypstFile failed: %v", err)
	}

	state, blocks, err := DeserializeTypstFile(data)
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}

	if !bytes.Equal(state, globalState) {
		t.Errorf("state mismatch")
	}

	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}

	if blocks[0].ID() != originalBlock.ID() {
		t.Errorf("ID mismatch: %s vs %s", originalBlock.ID(), blocks[0].ID())
	}
}
