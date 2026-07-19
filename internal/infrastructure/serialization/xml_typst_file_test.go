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

func newTestTypstFile(t *testing.T, blocks []block.Block) *file.TypstFile {
	t.Helper()
	f, err := file.NewTypstFile(uuid.New(), uuid.New(), "document.typ", blocks, time.Now())
	if err != nil {
		t.Fatalf("Failed to create TypstFile: %v", err)
	}
	return f
}

func TestSerializeTypstFile(t *testing.T) {
	t.Parallel()

	state1 := []byte("state-1")
	state2 := []byte("state-2")
	b1, _ := block.NewBlock(testBlockID1, "Введение", state1, "= Введение\nТекст введения")
	b2, _ := block.NewBlock(testBlockID2, "Глава 1", state2, "= Глава 1\nТекст главы")

	f := newTestTypstFile(t, []block.Block{b1, b2})

	data, err := SerializeTypstFile(f)
	if err != nil {
		t.Fatalf("SerializeTypstFile failed: %v", err)
	}

	xmlStr := string(data)

	if !strings.Contains(xmlStr, "<file>") {
		t.Errorf("Expected <file> root element, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `id="10000000-0000-0000-0000-000000000001"`) {
		t.Errorf("Expected first block id, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `id="20000000-0000-0000-0000-000000000002"`) {
		t.Errorf("Expected second block id, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `name="Введение"`) {
		t.Errorf("Expected block name 'Введение', got:\n%s", xmlStr)
	}

	expectedState1 := base64.StdEncoding.EncodeToString(state1)
	if !strings.Contains(xmlStr, `state="`+expectedState1+`"`) {
		t.Errorf("Expected base64 state for block 1, got:\n%s", xmlStr)
	}
}

func TestDeserializeTypstFile(t *testing.T) {
	t.Parallel()

	state1 := []byte("state-1")
	state2 := []byte("state-2")
	state1Base64 := base64.StdEncoding.EncodeToString(state1)
	state2Base64 := base64.StdEncoding.EncodeToString(state2)

	xmlData := `<file>
    <block id="10000000-0000-0000-0000-000000000001" name="Введение" state="` + state1Base64 + `">= Введение
Текст введения</block>
    <block id="20000000-0000-0000-0000-000000000002" name="Глава 1" state="` + state2Base64 + `">= Глава 1
Текст главы</block>
</file>`

	blocks, err := DeserializeTypstFile([]byte(xmlData))
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].ID() != testBlockID1 {
		t.Errorf("Block 0: expected ID %s, got %s", testBlockID1, blocks[0].ID())
	}
	if blocks[0].Name() != "Введение" {
		t.Errorf("Block 0: expected Name 'Введение', got %q", blocks[0].Name())
	}
	if !bytes.Equal(blocks[0].State(), state1) {
		t.Errorf("Block 0: State mismatch")
	}
	if blocks[0].Content() != "= Введение\nТекст введения" {
		t.Errorf("Block 0: expected content, got %q", blocks[0].Content())
	}

	if blocks[1].ID() != testBlockID2 {
		t.Errorf("Block 1: expected ID %s, got %s", testBlockID2, blocks[1].ID())
	}
}

func TestSerializeDeserializeTypstFile_Roundup(t *testing.T) {
	t.Parallel()

	state := []byte{0x00, 0xFF, 0xAB, 0xCD, 0xEF}
	original, _ := block.NewBlock(testBlockID1, "Test Block", state, "= Test\nContent here")

	f := newTestTypstFile(t, []block.Block{original})

	data, err := SerializeTypstFile(f)
	if err != nil {
		t.Fatalf("SerializeTypstFile failed: %v", err)
	}

	blocks, err := DeserializeTypstFile(data)
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}

	if blocks[0].ID() != original.ID() {
		t.Errorf("ID mismatch: %s vs %s", original.ID(), blocks[0].ID())
	}
	if blocks[0].Name() != original.Name() {
		t.Errorf("Name mismatch: %q vs %q", original.Name(), blocks[0].Name())
	}
	if !bytes.Equal(blocks[0].State(), original.State()) {
		t.Errorf("State mismatch")
	}
	if blocks[0].Content() != original.Content() {
		t.Errorf("Content mismatch: %q vs %q", original.Content(), blocks[0].Content())
	}
}

func TestDeserializeTypstFile_EmptyDocument(t *testing.T) {
	t.Parallel()

	blocks, err := DeserializeTypstFile([]byte(`<file></file>`))
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(blocks))
	}
}

func TestDeserializeTypstFile_InvalidXML(t *testing.T) {
	t.Parallel()

	_, err := DeserializeTypstFile([]byte(`<file><block not valid xml`))
	if err == nil {
		t.Error("Expected error for invalid XML, got nil")
	}
}

func TestSerializeTypstFile_NoBlocks(t *testing.T) {
	t.Parallel()

	f := newTestTypstFile(t, nil)

	data, err := SerializeTypstFile(f)
	if err != nil {
		t.Fatalf("SerializeTypstFile failed: %v", err)
	}
	if !strings.Contains(string(data), "<file>") {
		t.Errorf("Expected <file> element even with no blocks, got:\n%s", string(data))
	}
}
