package serialization

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/file"
)

func newTestTypstFile(t *testing.T, state []byte) *file.TypstFile {
	t.Helper()
	f, err := file.NewTypstFile(uuid.New(), uuid.New(), "document.typ", state, time.Now())
	if err != nil {
		t.Fatalf("Failed to create TypstFile: %v", err)
	}
	return f
}

func TestSerializeTypstFile(t *testing.T) {
	t.Parallel()

	globalState := []byte("global-crdt-state")
	f := newTestTypstFile(t, globalState)

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
}

func TestDeserializeTypstFile(t *testing.T) {
	t.Parallel()

	globalState := []byte("global-crdt-state")
	stateB64 := base64.StdEncoding.EncodeToString(globalState)

	xmlData := `<file state="` + stateB64 + `"></file>`

	state, err := DeserializeTypstFile([]byte(xmlData))
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}

	if !bytes.Equal(state, globalState) {
		t.Errorf("expected state %s, got %s", globalState, state)
	}
}

func TestSerializeDeserializeTypstFile_Roundtrip(t *testing.T) {
	t.Parallel()

	globalState := []byte{0x01, 0x02, 0x03, 0x04}
	f := newTestTypstFile(t, globalState)

	data, err := SerializeTypstFile(f)
	if err != nil {
		t.Fatalf("SerializeTypstFile failed: %v", err)
	}

	state, err := DeserializeTypstFile(data)
	if err != nil {
		t.Fatalf("DeserializeTypstFile failed: %v", err)
	}

	if !bytes.Equal(state, globalState) {
		t.Errorf("state mismatch")
	}
}
