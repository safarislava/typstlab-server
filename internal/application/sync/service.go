package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainMeta "github.com/safarislava/typstlab-server/internal/domain/metadata"
)

// Action defines the synchronization action required for a file.
type Action string

const (
	ActionDownload     Action = "download"
	ActionUpload       Action = "upload"
	ActionApplyChanges Action = "apply_changes"
)

// Instruction represents a command to be executed by the client to synchronize file content.
type Instruction struct {
	Action Action
	FileID uuid.UUID
	Delta  []byte
}

// SyncInitRequest contains the parameters needed for full project synchronization.
type SyncInitRequest struct {
	ProjectID           uuid.UUID
	MetadataStateVector []byte
	ContentVectors      map[uuid.UUID][]byte
}

// SyncInitResponse contains the metadata delta and content instructions produced by initial synchronization.
type SyncInitResponse struct {
	MetadataDelta []byte
	Instructions  []Instruction
}

// ApplyFileChangesRequest specifies a CRDT delta to apply to a single file.
type ApplyFileChangesRequest struct {
	FileID uuid.UUID
	Delta  []byte
}

// CRDTEngine unifies all CRDT-related operations for files and metadata.
type CRDTEngine interface {
	MergeFile(state, delta []byte) ([]byte, error)
	ComputeDelta(serverState, clientStateVector []byte) ([]byte, error)
	SyncMetadata(
		projectID uuid.UUID,
		currentMeta *domainMeta.Metadata,
		clientDelta []byte,
		clientStateVector []byte,
	) (metadataDelta []byte, meta *domainMeta.Metadata, err error)
}

// FileService defines operations on general files needed during synchronization.
type FileService interface {
	ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
	Save(ctx context.Context, f domainFile.File) error
}

// TypstFileService defines operations on Typst files needed during synchronization.
type TypstFileService interface {
	GetByID(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	Save(ctx context.Context, f *domainFile.TypstFile) error
}

// MetadataRepository defines the contract for loading project metadata entries.
type MetadataRepository interface {
	FindEntriesByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domainEntry.Entry, error)
}

// Service orchestrates full and partial synchronization of project metadata and files.
type Service struct {
	metaRepo    MetadataRepository
	fileService FileService
	typstFile   TypstFileService
	crdtEngine  CRDTEngine
}

// NewService creates a new unified synchronization Service.
func NewService(
	metaRepo MetadataRepository,
	fileService FileService,
	typstFile TypstFileService,
	crdtEngine CRDTEngine,
) *Service {
	return &Service{
		metaRepo:    metaRepo,
		fileService: fileService,
		typstFile:   typstFile,
		crdtEngine:  crdtEngine,
	}
}

// SyncInit coordinates complete project metadata and file content synchronization.
func (s *Service) SyncInit(ctx context.Context, req SyncInitRequest) (*SyncInitResponse, error) {
	currentMeta, err := s.loadProjectMetadata(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("load project metadata: %w", err)
	}

	metaDelta, updatedMeta, err := s.syncMetadataState(req.ProjectID, currentMeta, req.MetadataStateVector)
	if err != nil {
		return nil, fmt.Errorf("sync metadata state: %w", err)
	}

	serverFiles, err := s.listServerFiles(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("list server files: %w", err)
	}

	if applyErr := s.applyFileMutations(ctx, serverFiles, updatedMeta); applyErr != nil {
		return nil, fmt.Errorf("apply file mutations: %w", applyErr)
	}

	instructions, err := s.generateInstructions(serverFiles, updatedMeta, req.ContentVectors)
	if err != nil {
		return nil, fmt.Errorf("generate instructions: %w", err)
	}

	return &SyncInitResponse{
		MetadataDelta: metaDelta,
		Instructions:  instructions,
	}, nil
}

// ApplyMetadataUpdate applies incremental project metadata CRDT deltas and mutates files without full content scan.
func (s *Service) ApplyMetadataUpdate(ctx context.Context, projectID uuid.UUID, delta []byte) error {
	if len(delta) == 0 {
		return errors.New("metadata delta cannot be empty")
	}

	currentMeta, err := s.loadProjectMetadata(ctx, projectID)
	if err != nil {
		return fmt.Errorf("load project metadata: %w", err)
	}

	_, updatedMeta, err := s.mergeMetadataDelta(projectID, currentMeta, delta)
	if err != nil {
		return fmt.Errorf("merge metadata delta: %w", err)
	}

	serverFiles, err := s.listServerFiles(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list server files: %w", err)
	}

	if applyErr := s.applyFileMutations(ctx, serverFiles, updatedMeta); applyErr != nil {
		return fmt.Errorf("apply file mutations: %w", applyErr)
	}

	return nil
}

// ApplyCRDTUpdate persists incremental CRDT file changes for a single file.
func (s *Service) ApplyCRDTUpdate(ctx context.Context, fileID uuid.UUID, delta []byte) error {
	if fileID == uuid.Nil {
		return errors.New("file_id is required")
	}
	if len(delta) == 0 {
		return errors.New("delta cannot be empty")
	}

	_, err := s.ApplyFileChanges(ctx, ApplyFileChangesRequest{
		FileID: fileID,
		Delta:  delta,
	})
	return err
}

// ApplyFileChanges applies a CRDT delta to a specific Typst file and saves the updated state.
func (s *Service) ApplyFileChanges(ctx context.Context, req ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	file, err := s.loadTypstFile(ctx, req.FileID)
	if err != nil {
		return nil, fmt.Errorf("load typst file: %w", err)
	}

	mergedState, err := s.mergeFileContent(file.State(), req.Delta)
	if err != nil {
		return nil, fmt.Errorf("merge file content: %w", err)
	}

	if err := s.persistTypstFileState(ctx, file, mergedState); err != nil {
		return nil, fmt.Errorf("persist typst file: %w", err)
	}

	return file, nil
}

func (s *Service) loadProjectMetadata(ctx context.Context, projectID uuid.UUID) (*domainMeta.Metadata, error) {
	entries, err := s.metaRepo.FindEntriesByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("find project metadata entries: %w", err)
	}
	meta, err := domainMeta.NewMetadata(projectID, entries)
	if err != nil {
		return nil, fmt.Errorf("construct metadata aggregate: %w", err)
	}
	return meta, nil
}

func (s *Service) syncMetadataState(
	projectID uuid.UUID,
	meta *domainMeta.Metadata,
	stateVector []byte,
) ([]byte, *domainMeta.Metadata, error) {
	delta, updatedMeta, err := s.crdtEngine.SyncMetadata(projectID, meta, nil, stateVector)
	if err != nil {
		return nil, nil, fmt.Errorf("sync metadata state vector: %w", err)
	}
	return delta, updatedMeta, nil
}

func (s *Service) mergeMetadataDelta(
	projectID uuid.UUID,
	meta *domainMeta.Metadata,
	delta []byte,
) ([]byte, *domainMeta.Metadata, error) {
	metaDelta, updatedMeta, err := s.crdtEngine.SyncMetadata(projectID, meta, delta, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("sync metadata delta: %w", err)
	}
	return metaDelta, updatedMeta, nil
}

func (s *Service) listServerFiles(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	files, err := s.fileService.ListFilesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	return files, nil
}

func (s *Service) loadTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error) {
	file, err := s.typstFile.GetByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("get typst file: %w", err)
	}
	return file, nil
}

func (s *Service) mergeFileContent(state, delta []byte) ([]byte, error) {
	merged, err := s.crdtEngine.MergeFile(state, delta)
	if err != nil {
		return nil, fmt.Errorf("merge file state: %w", err)
	}
	return merged, nil
}

func (s *Service) persistTypstFileState(ctx context.Context, file *domainFile.TypstFile, state []byte) error {
	if err := file.UpdateState(state); err != nil {
		return fmt.Errorf("update typst file state: %w", err)
	}
	if err := s.typstFile.Save(ctx, file); err != nil {
		return fmt.Errorf("save typst file: %w", err)
	}
	return nil
}

func (s *Service) applyFileMutations(
	ctx context.Context,
	serverFiles []domainFile.File,
	meta *domainMeta.Metadata,
) error {
	serverByID := indexFilesByID(serverFiles)

	for _, entry := range meta.Entries() {
		sf, exists := serverByID[entry.ID()]
		if !exists {
			continue
		}

		if err := s.mutateFileIfNeeded(ctx, sf, entry); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) mutateFileIfNeeded(ctx context.Context, sf domainFile.File, entry *domainEntry.Entry) error {
	if entry.IsDeleted() {
		if err := s.fileService.DeleteFile(ctx, entry.ID()); err != nil {
			return fmt.Errorf("delete file: %w", err)
		}
		return nil
	}

	if entry.Name() != sf.Name() {
		return s.renameFile(ctx, sf, entry.Name())
	}

	return nil
}

func (s *Service) renameFile(ctx context.Context, sf domainFile.File, newName string) error {
	if err := sf.Rename(newName); err != nil {
		return fmt.Errorf("rename file %s: %w", sf.ID(), err)
	}

	if err := s.fileService.Save(ctx, sf); err != nil {
		return fmt.Errorf("save renamed file %s: %w", sf.ID(), err)
	}

	return nil
}

func (s *Service) generateInstructions(
	serverFiles []domainFile.File,
	meta *domainMeta.Metadata,
	contentVectors map[uuid.UUID][]byte,
) ([]Instruction, error) {
	serverByID := indexFilesByID(serverFiles)

	uploadInstructions := s.buildUploadInstructions(meta, serverByID)
	serverInstructions, err := s.buildServerFileInstructions(serverFiles, meta, contentVectors)
	if err != nil {
		return nil, err
	}

	return append(uploadInstructions, serverInstructions...), nil
}

func (s *Service) buildUploadInstructions(
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

func (s *Service) buildServerFileInstructions(
	serverFiles []domainFile.File,
	meta *domainMeta.Metadata,
	contentVectors map[uuid.UUID][]byte,
) ([]Instruction, error) {
	var instructions []Instruction

	for _, sf := range serverFiles {
		entry, _ := meta.Get(sf.ID())
		inst, err := s.resolveFileInstruction(sf, entry, contentVectors[sf.ID()])
		if err != nil {
			return nil, err
		}
		if inst != nil {
			instructions = append(instructions, *inst)
		}
	}

	return instructions, nil
}

func (s *Service) resolveFileInstruction(
	sf domainFile.File,
	entry *domainEntry.Entry,
	clientVector []byte,
) (*Instruction, error) {
	if entry != nil && entry.IsDeleted() {
		return nil, nil
	}

	if len(clientVector) == 0 {
		return &Instruction{
			Action: ActionDownload,
			FileID: sf.ID(),
		}, nil
	}

	return s.calculateTypstDeltaInstruction(sf, clientVector)
}

func (s *Service) calculateTypstDeltaInstruction(sf domainFile.File, clientVector []byte) (*Instruction, error) {
	typstFile, ok := sf.(*domainFile.TypstFile)
	if !ok {
		return nil, nil
	}

	delta, err := s.crdtEngine.ComputeDelta(typstFile.State(), clientVector)
	if err != nil {
		return nil, fmt.Errorf("compute delta for file %s: %w", sf.ID(), err)
	}

	if len(delta) == 0 {
		return nil, nil
	}

	return &Instruction{
		Action: ActionApplyChanges,
		FileID: sf.ID(),
		Delta:  delta,
	}, nil
}

func indexFilesByID(files []domainFile.File) map[uuid.UUID]domainFile.File {
	byID := make(map[uuid.UUID]domainFile.File, len(files))
	for _, f := range files {
		byID[f.ID()] = f
	}
	return byID
}
