package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	domainBlock "github.com/safarislava/typstlab-server/internal/domain/block"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	"github.com/safarislava/typstlab-server/internal/infrastructure/serialization"
)

type FileHandler struct {
	fileService fileApp.UseCase
}

func NewFileHandler(fileService fileApp.UseCase) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

func (h *FileHandler) writeJSONFileResponse(w http.ResponseWriter, f domainFile.File, status int) {
	resp := JSONFileResponse{
		ID:        f.ID().String(),
		ProjectID: f.ProjectID().String(),
		Name:      f.Name(),
		Type:      string(f.Type()),
		UpdatedAt: f.UpdatedAt().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *FileHandler) writeJSONTypstFileResponse(w http.ResponseWriter, f *domainFile.TypstFile, status int) {
	blocks := f.Blocks()
	jsonBlocks := make([]JSONBlockResponse, len(blocks))
	for i, b := range blocks {
		jsonBlocks[i] = JSONBlockResponse{
			ID:      b.ID().String(),
			Name:    b.Name(),
			Content: b.Content(),
		}
	}

	resp := JSONTypstFileResponse{
		ID:        f.ID().String(),
		ProjectID: f.ProjectID().String(),
		Name:      f.Name(),
		Type:      string(f.Type()),
		State:     f.State(),
		Blocks:    jsonBlocks,
		UpdatedAt: f.UpdatedAt().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *FileHandler) parseUploadRequest(r *http.Request) (id uuid.UUID, name string, content []byte, err error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return h.parseMultipartUploadRequest(r)
	}
	return h.parseJSONUploadRequest(r)
}

func (h *FileHandler) parseMultipartUploadRequest(r *http.Request) (id uuid.UUID, name string, content []byte, err error) {
	if err = r.ParseMultipartForm(10 << 20); err != nil {
		return uuid.Nil, "", nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}
	idStr := r.FormValue("id")
	id, err = uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, "", nil, fmt.Errorf("invalid or missing file id: %w", err)
	}
	name = r.FormValue("name")
	file, header, fileErr := r.FormFile("file")
	if fileErr == nil {
		defer func() { _ = file.Close() }()
		if name == "" {
			name = header.Filename
		}
		content, err = io.ReadAll(file)
		if err != nil {
			return uuid.Nil, "", nil, fmt.Errorf("failed to read multipart file: %w", err)
		}
	}
	return id, name, content, nil
}

func (h *FileHandler) parseJSONUploadRequest(r *http.Request) (id uuid.UUID, name string, content []byte, err error) {
	var jsonReq jsonUploadFileRequest
	if err = json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		return uuid.Nil, "", nil, fmt.Errorf("failed to parse json request: %w", err)
	}
	id, err = uuid.Parse(jsonReq.ID)
	if err != nil {
		return uuid.Nil, "", nil, fmt.Errorf("invalid or missing file id: %w", err)
	}
	return id, jsonReq.Name, jsonReq.Content, nil
}

type jsonUploadFileRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content []byte `json:"content"`
}

type JSONFileResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	UpdatedAt string `json:"updated_at"`
}

type JSONBlockResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type JSONTypstFileResponse struct {
	ID        string              `json:"id"`
	ProjectID string              `json:"project_id"`
	Name      string              `json:"name"`
	Type      string              `json:"type"`
	State     []byte              `json:"state"`
	Blocks    []JSONBlockResponse `json:"blocks"`
	UpdatedAt string              `json:"updated_at"`
}

type JSONBinaryFileResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Size      int    `json:"size"`
	UpdatedAt string `json:"updated_at"`
}

type jsonApplyFileChangesRequest struct {
	Delta []byte `json:"delta"`
}

func (h *FileHandler) initTypstFile(content []byte) (state []byte, blocks []domainBlock.Block, err error) {
	if len(content) == 0 {
		return nil, nil, nil
	}
	state, blocks, err = serialization.DeserializeTypstFile(content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize typst file: %w", err)
	}
	return state, blocks, nil
}

func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	p, ok := ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	id, name, content, err := h.parseUploadRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("File name cannot be empty"))
		return
	}

	var f domainFile.File
	if strings.HasSuffix(name, ".typxml") {
		state, blocks, errInit := h.initTypstFile(content)
		if errInit != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(errInit.Error()))
			return
		}

		req := fileApp.UploadTypstFileRequest{
			ID:        id,
			ProjectID: p.ID(),
			Name:      name,
			State:     state,
			Blocks:    blocks,
		}
		f, err = h.fileService.UploadTypstFile(r.Context(), &req)
	} else {
		req := fileApp.UploadBinaryFileRequest{
			ID:        id,
			ProjectID: p.ID(),
			Name:      name,
			Content:   content,
		}
		f, err = h.fileService.UploadBinaryFile(r.Context(), &req)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	h.writeJSONFileResponse(w, f, http.StatusCreated)
}

func (h *FileHandler) ListProjectFiles(w http.ResponseWriter, r *http.Request) {
	p, ok := ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	files, err := h.fileService.ListFilesByProject(r.Context(), p.ID())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	resp := make([]JSONFileResponse, len(files))
	for i, f := range files {
		resp[i] = JSONFileResponse{
			ID:        f.ID().String(),
			ProjectID: f.ProjectID().String(),
			Name:      f.Name(),
			Type:      string(f.Type()),
			UpdatedAt: f.UpdatedAt().Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *FileHandler) GetTypstFile(w http.ResponseWriter, r *http.Request) {
	f, ok := FileFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("File not found in context"))
		return
	}

	tf, isTypst := f.(*domainFile.TypstFile)
	if !isTypst {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Not a Typst file"))
		return
	}

	h.writeJSONTypstFileResponse(w, tf, http.StatusOK)
}

func (h *FileHandler) GetBinaryFile(w http.ResponseWriter, r *http.Request) {
	f, ok := FileFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("File not found in context"))
		return
	}

	bf, isBinary := f.(*domainFile.BinaryFile)
	if !isBinary {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Not a binary file"))
		return
	}

	resp := JSONBinaryFileResponse{
		ID:        bf.ID().String(),
		ProjectID: bf.ProjectID().String(),
		Name:      bf.Name(),
		Type:      string(bf.Type()),
		Size:      len(bf.Content()),
		UpdatedAt: bf.UpdatedAt().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *FileHandler) GetBinaryFileRaw(w http.ResponseWriter, r *http.Request) {
	f, ok := FileFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("File not found in context"))
		return
	}

	bf, isBinary := f.(*domainFile.BinaryFile)
	if !isBinary {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Not a binary file"))
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+bf.Name())
	_, _ = w.Write(bf.Content())
}

func (h *FileHandler) ApplyFileChanges(w http.ResponseWriter, r *http.Request) {
	f, ok := FileFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("File not found in context"))
		return
	}

	tf, isTypst := f.(*domainFile.TypstFile)
	if !isTypst {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Not a Typst file"))
		return
	}

	var jsonReq jsonApplyFileChangesRequest
	if errJSON := json.NewDecoder(r.Body).Decode(&jsonReq); errJSON != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid request body"))
		return
	}

	req := fileApp.ApplyFileChangesRequest{
		FileID: tf.ID(),
		Delta:  jsonReq.Delta,
	}

	updatedFile, err := h.fileService.ApplyFileChanges(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	h.writeJSONTypstFileResponse(w, updatedFile, http.StatusOK)
}

func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	p, okP := ProjectFromContext(r.Context())
	f, okF := FileFromContext(r.Context())
	if !okP || !okF {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Context requirements not met"))
		return
	}

	if f.ProjectID() != p.ID() {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("File does not belong to the specified project"))
		return
	}

	if err := h.fileService.DeleteFile(r.Context(), f.ID()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Failed to delete file: " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
