package serialization

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"

	"github.com/safarislava/typstlab-server/internal/domain/file"
)

type xmlTypstFile struct {
	XMLName xml.Name `xml:"file"`
	State   string   `xml:"state,attr"`
}

func SerializeTypstFile(f *file.TypstFile) ([]byte, error) {
	typstFile := xmlTypstFile{
		State: base64.StdEncoding.EncodeToString(f.State()),
	}

	serialized, err := xml.MarshalIndent(typstFile, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize typst file: %w", err)
	}

	return serialized, nil
}

func DeserializeTypstFile(data []byte) ([]byte, error) {
	var doc xmlTypstFile
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize typst file: %w", err)
	}

	decodedState, err := base64.StdEncoding.DecodeString(doc.State)
	if err != nil {
		return nil, fmt.Errorf("failed to decode global file state: %w", err)
	}

	return decodedState, nil
}
