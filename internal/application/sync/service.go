package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
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

type Service struct {
	repo fileApp.Repository
}

func NewService(repo fileApp.Repository) UseCase {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Sync(ctx context.Context, projectID uuid.UUID, req *Request) (*Response, error) {
	serverFiles, err := s.repo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project files: %w", err)
	}

	filesByID, filesByName := s.buildLookupMaps(serverFiles)
	var instructions []Instruction
	clientIDs := make(map[uuid.UUID]bool)

	for _, clientFile := range req.Files {
		clientIDs[clientFile.ID] = true

		fileInstructions, err := s.processClientFile(ctx, clientFile, filesByID, filesByName)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, fileInstructions...)
	}

	missingInstructions := s.processMissingFiles(serverFiles, clientIDs)
	instructions = append(instructions, missingInstructions...)

	return &Response{
		Instructions: instructions,
	}, nil
}

func (s *Service) processClientFile(ctx context.Context, clientFile FileRequest, filesByID map[uuid.UUID]domainFile.File, filesByName map[string]domainFile.File) ([]Instruction, error) {
	if serverFile, exists := filesByID[clientFile.ID]; exists {
		return s.processServerFileExists(clientFile, serverFile)
	}

	isDeleted, err := s.repo.IsDeleted(ctx, clientFile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check file deletion status: %w", err)
	}

	if isDeleted {
		return []Instruction{
			{
				Action: ActionDelete,
				FileID: clientFile.ID,
			},
		}, nil
	}

	return s.processOfflineFile(clientFile, filesByName), nil
}

func (s *Service) processMissingFiles(serverFiles []domainFile.File, clientIDs map[uuid.UUID]bool) []Instruction {
	var instructions []Instruction
	for _, serverFile := range serverFiles {
		if !clientIDs[serverFile.ID()] {
			instructions = append(instructions, Instruction{
				Action: ActionDownload,
				FileID: serverFile.ID(),
			})
		}
	}
	return instructions
}

func (s *Service) buildLookupMaps(files []domainFile.File) (filesByID map[uuid.UUID]domainFile.File, filesByName map[string]domainFile.File) {
	filesByID = make(map[uuid.UUID]domainFile.File)
	filesByName = make(map[string]domainFile.File)
	for _, serverFile := range files {
		filesByID[serverFile.ID()] = serverFile
		filesByName[serverFile.Name()] = serverFile
	}
	return filesByID, filesByName
}

func (s *Service) processOfflineFile(clientFile FileRequest, filesByName map[string]domainFile.File) []Instruction {
	var instructions []Instruction
	newName := clientFile.Name
	if _, nameConflict := filesByName[clientFile.Name]; nameConflict {
		ext := ""
		base := clientFile.Name
		if idx := strings.LastIndex(clientFile.Name, "."); idx != -1 {
			base = clientFile.Name[:idx]
			ext = clientFile.Name[idx:]
		}
		newName = base + "_conflict" + ext
	}

	if newName != clientFile.Name {
		instructions = append(instructions, Instruction{
			Action:  ActionRename,
			FileID:  clientFile.ID,
			NewName: newName,
		})
	}

	instructions = append(instructions, Instruction{
		Action: ActionUpload,
		FileID: clientFile.ID,
	})

	return instructions
}

func (s *Service) processServerFileExists(clientFile FileRequest, serverFile domainFile.File) ([]Instruction, error) {
	var instructions []Instruction
	if clientFile.Name != serverFile.Name() {
		instructions = append(instructions, Instruction{
			Action:  ActionRename,
			FileID:  clientFile.ID,
			NewName: serverFile.Name(),
		})
	}

	if serverFile.Type() != domainFile.TypeTypst || len(clientFile.State) == 0 {
		return instructions, nil
	}

	typstFile, ok := serverFile.(*domainFile.TypstFile)
	if !ok {
		return instructions, nil
	}

	delta, err := s.computeDelta(typstFile, clientFile.State)
	if err != nil {
		return nil, err
	}

	if len(delta) > 0 {
		instructions = append(instructions, Instruction{
			Action: ActionApplyChanges,
			FileID: clientFile.ID,
			Delta:  delta,
		})
	}

	return instructions, nil
}

func (s *Service) computeDelta(typstFile *domainFile.TypstFile, clientState []byte) ([]byte, error) {
	doc := crdt.New()
	if len(typstFile.State()) > 0 {
		if err := doc.ApplyUpdate(typstFile.State()); err != nil {
			return nil, fmt.Errorf("failed to apply server state update: %w", err)
		}
	}

	stateVector, err := crdt.DecodeStateVectorV1(clientState)
	if err != nil {
		return nil, fmt.Errorf("failed to decode client state vector: %w", err)
	}

	return crdt.EncodeStateAsUpdateV1(doc, stateVector), nil
}
