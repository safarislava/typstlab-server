package sync

import (
	"context"

	"github.com/google/uuid"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type FileRequest struct {
	ID    uuid.UUID
	Name  string
	Type  domainFile.Type
	State []byte
}

type Request struct {
	Files []FileRequest
}

type Action string

const (
	ActionDownload     Action = "download"
	ActionUpload       Action = "upload"
	ActionRename       Action = "rename"
	ActionDelete       Action = "delete"
	ActionApplyChanges Action = "apply_changes"
)

type Instruction struct {
	Action  Action
	FileID  uuid.UUID
	NewName string // для rename
	Delta   []byte // для apply_changes
}

type Response struct {
	Instructions []Instruction
}

type UseCase interface {
	Sync(ctx context.Context, projectID uuid.UUID, req *Request) (*Response, error)
}
