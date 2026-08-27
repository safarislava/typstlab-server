package sync

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/reearth/ygo/crdt"

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

type Repository interface {
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	IsDeleted(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Sync(ctx context.Context, projectID uuid.UUID, req *Request) (*Response, error) {
	serverFiles, err := s.repo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project files: %w", err)
	}

	matching, clientOnly, serverOnly := s.categorizeFiles(req.Files, serverFiles)

	matchingInstructions, err := s.processMatchingFiles(matching)
	if err != nil {
		return nil, err
	}

	clientOnlyInstructions, err := s.processClientOnlyFiles(ctx, clientOnly, serverFiles)
	if err != nil {
		return nil, err
	}

	serverOnlyInstructions := s.processServerOnlyFiles(serverOnly)

	instructions := slices.Concat(matchingInstructions, clientOnlyInstructions, serverOnlyInstructions)

	return &Response{
		Instructions: instructions,
	}, nil
}

type fileMatch struct {
	client FileRequest
	server domainFile.File
}

func (s *Service) categorizeFiles(clientFiles []FileRequest, serverFiles []domainFile.File) (
	matching []*fileMatch,
	clientOnly []FileRequest,
	serverOnly []domainFile.File,
) {
	serverByID := make(map[uuid.UUID]domainFile.File, len(serverFiles))
	for _, sf := range serverFiles {
		serverByID[sf.ID()] = sf
	}

	clientIDs := make(map[uuid.UUID]bool, len(clientFiles))
	for _, cf := range clientFiles {
		clientIDs[cf.ID] = true
		if sf, exists := serverByID[cf.ID]; exists {
			matching = append(matching, &fileMatch{client: cf, server: sf})
		} else {
			clientOnly = append(clientOnly, cf)
		}
	}

	for _, sf := range serverFiles {
		if !clientIDs[sf.ID()] {
			serverOnly = append(serverOnly, sf)
		}
	}

	return matching, clientOnly, serverOnly
}

func (s *Service) processMatchingFiles(matches []*fileMatch) ([]Instruction, error) {
	var instructions []Instruction

	for _, match := range matches {
		matchInstructions, err := s.processSingleMatch(match)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, matchInstructions...)
	}

	return instructions, nil
}

func (s *Service) processSingleMatch(match *fileMatch) ([]Instruction, error) {
	var instructions []Instruction

	if match.client.Name != match.server.Name() {
		instructions = append(instructions, Instruction{
			Action:  ActionRename,
			FileID:  match.client.ID,
			NewName: match.server.Name(),
		})
	}

	deltaInst, err := s.generateDeltaInstruction(match)
	if err != nil {
		return nil, err
	}
	if deltaInst != nil {
		instructions = append(instructions, *deltaInst)
	}

	return instructions, nil
}

func (s *Service) generateDeltaInstruction(match *fileMatch) (*Instruction, error) {
	if len(match.client.State) == 0 {
		return nil, nil
	}

	typstFile, isTypst := match.server.(*domainFile.TypstFile)
	if !isTypst {
		return nil, nil
	}

	delta, err := s.computeDelta(typstFile, match.client.State)
	if err != nil {
		return nil, err
	}

	if len(delta) == 0 {
		return nil, nil
	}

	return &Instruction{
		Action: ActionApplyChanges,
		FileID: match.client.ID,
		Delta:  delta,
	}, nil
}

func (s *Service) processClientOnlyFiles(ctx context.Context, clientFiles []FileRequest, serverFiles []domainFile.File) ([]Instruction, error) {
	existingNames := buildNameSet(serverFiles)
	var instructions []Instruction

	for _, cf := range clientFiles {
		fileInstructions, err := s.processSingleClientOnlyFile(ctx, cf, existingNames)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, fileInstructions...)
	}

	return instructions, nil
}

func (s *Service) processSingleClientOnlyFile(ctx context.Context, cf FileRequest, existingNames map[string]bool) ([]Instruction, error) {
	isDeleted, err := s.repo.IsDeleted(ctx, cf.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check deletion status for %s: %w", cf.ID, err)
	}
	if isDeleted {
		return []Instruction{{Action: ActionDelete, FileID: cf.ID}}, nil
	}

	return s.createUploadInstructions(cf, existingNames), nil
}

func (s *Service) createUploadInstructions(cf FileRequest, existingNames map[string]bool) []Instruction {
	var instructions []Instruction

	if resolvedName := resolveConflictName(cf.Name, existingNames); resolvedName != cf.Name {
		instructions = append(instructions, Instruction{
			Action:  ActionRename,
			FileID:  cf.ID,
			NewName: resolvedName,
		})
	}

	return append(instructions, Instruction{
		Action: ActionUpload,
		FileID: cf.ID,
	})
}

func buildNameSet(files []domainFile.File) map[string]bool {
	names := make(map[string]bool, len(files))
	for _, f := range files {
		names[f.Name()] = true
	}
	return names
}

func (s *Service) processServerOnlyFiles(serverFiles []domainFile.File) []Instruction {
	instructions := make([]Instruction, 0, len(serverFiles))
	for _, sf := range serverFiles {
		instructions = append(instructions, Instruction{
			Action: ActionDownload,
			FileID: sf.ID(),
		})
	}
	return instructions
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

func resolveConflictName(name string, existingNames map[string]bool) string {
	if !existingNames[name] {
		return name
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s_conflict%s", base, ext)
}
