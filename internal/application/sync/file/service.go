package file

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/block"
	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

type Action string

const (
	ActionDownload     Action = "download"
	ActionUpload       Action = "upload"
	ActionApplyChanges Action = "apply_changes"
)

type Instruction struct {
	Action Action
	FileID uuid.UUID
	Delta  []byte
}

type ApplyFileChangesRequest struct {
	FileID uuid.UUID
	Delta  []byte
}

type FileManager interface {
	GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	SaveTypstFile(ctx context.Context, f *domainFile.TypstFile) error
	ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	RenameFile(ctx context.Context, fileID uuid.UUID, newName string) error
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
}

type FileMerger interface {
	MergeFile(state, delta []byte) (newState []byte, updatedBlocks []block.Block, err error)
}

type DeltaCalculator interface {
	ComputeDelta(serverState, clientStateVector []byte) ([]byte, error)
}

type Service struct {
	fileManager     FileManager
	fileMerger      FileMerger
	deltaCalculator DeltaCalculator
}

func NewService(
	fileManager FileManager,
	fileMerger FileMerger,
	deltaCalculator DeltaCalculator,
) *Service {
	return &Service{
		fileManager:     fileManager,
		fileMerger:      fileMerger,
		deltaCalculator: deltaCalculator,
	}
}

func (s *Service) ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	files, err := s.fileManager.ListFilesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project files: %w", err)
	}
	return files, nil
}

func (s *Service) ApplyFileChanges(ctx context.Context, req ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	f, err := s.fileManager.GetTypstFile(ctx, req.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find typst file: %w", err)
	}

	state, blocks, err := s.fileMerger.MergeFile(f.State(), req.Delta)
	if err != nil {
		return nil, fmt.Errorf("failed to merge file delta: %w", err)
	}

	if err := f.UpdateState(state, blocks); err != nil {
		return nil, fmt.Errorf("failed to update typst file aggregate state: %w", err)
	}

	if err := s.fileManager.SaveTypstFile(ctx, f); err != nil {
		return nil, fmt.Errorf("failed to save updated typst file: %w", err)
	}

	return f, nil
}

func (s *Service) ApplyMetadataMutations(
	ctx context.Context,
	serverFiles []domainFile.File,
	meta *domainMeta.Metadata,
) error {
	serverByID := make(map[uuid.UUID]domainFile.File, len(serverFiles))
	for _, sf := range serverFiles {
		serverByID[sf.ID()] = sf
	}

	for _, entry := range meta.Entries() {
		sf, exists := serverByID[entry.ID()]
		if !exists {
			continue
		}

		if err := s.applySingleEntryMutation(ctx, sf, entry); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) applySingleEntryMutation(
	ctx context.Context,
	sf domainFile.File,
	entry *domainEntry.Entry,
) error {
	if entry.IsDeleted() {
		if err := s.fileManager.DeleteFile(ctx, entry.ID()); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", entry.ID(), err)
		}
		return nil
	}

	if entry.Name() != sf.Name() {
		if err := s.fileManager.RenameFile(ctx, entry.ID(), entry.Name()); err != nil {
			return fmt.Errorf("failed to rename file %s: %w", entry.ID(), err)
		}
	}

	return nil
}

func (s *Service) GenerateContentInstructions(
	serverFiles []domainFile.File,
	meta *domainMeta.Metadata,
	contentVectors map[uuid.UUID][]byte,
) ([]Instruction, error) {
	serverByID := make(map[uuid.UUID]domainFile.File, len(serverFiles))
	for _, sf := range serverFiles {
		serverByID[sf.ID()] = sf
	}

	uploadInst := s.findUploadInstructions(meta, serverByID)

	serverInst, err := s.findServerFileInstructions(serverFiles, meta, contentVectors)
	if err != nil {
		return nil, err
	}

	return append(uploadInst, serverInst...), nil
}

func (s *Service) findUploadInstructions(
	meta *domainMeta.Metadata,
	serverByID map[uuid.UUID]domainFile.File,
) []Instruction {
	var instructions []Instruction
	for _, entry := range meta.ActiveEntries() {
		if _, exists := serverByID[entry.ID()]; !exists {
			instructions = append(instructions, Instruction{
				Action: ActionUpload,
				FileID: entry.ID(),
			})
		}
	}
	return instructions
}

func (s *Service) findServerFileInstructions(
	serverFiles []domainFile.File,
	meta *domainMeta.Metadata,
	contentVectors map[uuid.UUID][]byte,
) ([]Instruction, error) {
	var instructions []Instruction

	for _, sf := range serverFiles {
		entry, _ := meta.Get(sf.ID())
		inst, err := s.processSingleServerFile(sf, entry, contentVectors[sf.ID()])
		if err != nil {
			return nil, err
		}
		if inst != nil {
			instructions = append(instructions, *inst)
		}
	}

	return instructions, nil
}

func (s *Service) processSingleServerFile(
	sf domainFile.File,
	entry *domainEntry.Entry,
	clientVec []byte,
) (*Instruction, error) {
	if entry != nil && entry.IsDeleted() {
		return nil, nil
	}

	if len(clientVec) == 0 {
		return &Instruction{
			Action: ActionDownload,
			FileID: sf.ID(),
		}, nil
	}

	return s.checkContentDelta(sf, clientVec)
}

func (s *Service) checkContentDelta(serverFile domainFile.File, clientVec []byte) (*Instruction, error) {
	if len(clientVec) == 0 {
		return nil, nil
	}

	typstFile, isTypst := serverFile.(*domainFile.TypstFile)
	if !isTypst {
		return nil, nil
	}

	delta, err := s.deltaCalculator.ComputeDelta(typstFile.State(), clientVec)
	if err != nil {
		return nil, fmt.Errorf("failed to compute delta: %w", err)
	}

	if len(delta) == 0 {
		return nil, nil
	}

	return &Instruction{
		Action: ActionApplyChanges,
		FileID: serverFile.ID(),
		Delta:  delta,
	}, nil
}
