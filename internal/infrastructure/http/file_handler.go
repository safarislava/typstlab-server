package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type FileHandler struct {
	fileService    fileApp.UseCase
	projectService projectApp.UseCase
}

func NewFileHandler(fileService fileApp.UseCase, projectService projectApp.UseCase) *FileHandler {
	return &FileHandler{
		fileService:    fileService,
		projectService: projectService,
	}
}

func (h *FileHandler) checkProjectMembership(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) bool {
	p, err := h.projectService.Get(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Project not found"))
		return false
	}

	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	if !p.HasUser(userID) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden"))
		return false
	}

	return true
}

func (h *FileHandler) authorizeProjectAccess(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid project ID"))
		return uuid.Nil, false
	}

	if !h.checkProjectMembership(w, r, projectID) {
		return uuid.Nil, false
	}

	return projectID, true
}

func (h *FileHandler) authorizeTypstFileAccess(w http.ResponseWriter, r *http.Request) (f *domainFile.TypstFile, ok bool) {
	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid file ID"))
		return nil, false
	}

	f, err = h.fileService.GetTypstFile(r.Context(), fileID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Typst file not found"))
		return nil, false
	}

	if !h.checkProjectMembership(w, r, f.ProjectID()) {
		return nil, false
	}

	return f, true
}

func (h *FileHandler) authorizeBinaryFileAccess(w http.ResponseWriter, r *http.Request) (f *domainFile.BinaryFile, ok bool) {
	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid file ID"))
		return nil, false
	}

	f, err = h.fileService.GetBinaryFile(r.Context(), fileID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Binary file not found"))
		return nil, false
	}

	if !h.checkProjectMembership(w, r, f.ProjectID()) {
		return nil, false
	}

	return f, true
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

func (h *FileHandler) parseBinaryFileRequest(r *http.Request) (name string, content []byte, err error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return h.parseMultipartBinaryFileRequest(r)
	}
	return h.parseJSONBinaryFileRequest(r)
}

func (h *FileHandler) parseMultipartBinaryFileRequest(r *http.Request) (name string, content []byte, err error) {
	if err = r.ParseMultipartForm(10 << 20); err != nil {
		return "", nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse multipart form file: %w", err)
	}
	defer func() { _ = file.Close() }()
	name = r.FormValue("name")
	if name == "" {
		name = header.Filename
	}
	content, err = io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read multipart file: %w", err)
	}
	return name, content, nil
}

func (h *FileHandler) parseJSONBinaryFileRequest(r *http.Request) (name string, content []byte, err error) {
	var jsonReq jsonCreateBinaryFileRequest
	if err = json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		return "", nil, fmt.Errorf("failed to parse json request: %w", err)
	}
	return jsonReq.Name, jsonReq.Content, nil
}

type jsonCreateTypstFileRequest struct {
	Name string `json:"name"`
}

type jsonCreateBinaryFileRequest struct {
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

func (h *FileHandler) CreateTypstFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.authorizeProjectAccess(w, r)
	if !ok {
		return
	}

	var jsonReq jsonCreateTypstFileRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid request body"))
		return
	}

	if jsonReq.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("File name cannot be empty"))
		return
	}

	req := fileApp.CreateTypstFileRequest{
		ProjectID: projectID,
		Name:      jsonReq.Name,
	}

	f, err := h.fileService.CreateTypstFile(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	h.writeJSONFileResponse(w, f, http.StatusCreated)
}

func (h *FileHandler) CreateBinaryFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.authorizeProjectAccess(w, r)
	if !ok {
		return
	}

	name, content, err := h.parseBinaryFileRequest(r)
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

	req := fileApp.CreateBinaryFileRequest{
		ProjectID: projectID,
		Name:      name,
		Content:   content,
	}

	f, err := h.fileService.CreateBinaryFile(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	h.writeJSONFileResponse(w, f, http.StatusCreated)
}

func (h *FileHandler) ListProjectFiles(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.authorizeProjectAccess(w, r)
	if !ok {
		return
	}

	files, err := h.fileService.ListFilesByProject(r.Context(), projectID)
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
	f, ok := h.authorizeTypstFileAccess(w, r)
	if !ok {
		return
	}
	h.writeJSONTypstFileResponse(w, f, http.StatusOK)
}

func (h *FileHandler) GetBinaryFile(w http.ResponseWriter, r *http.Request) {
	f, ok := h.authorizeBinaryFileAccess(w, r)
	if !ok {
		return
	}

	resp := JSONBinaryFileResponse{
		ID:        f.ID().String(),
		ProjectID: f.ProjectID().String(),
		Name:      f.Name(),
		Type:      string(f.Type()),
		Size:      len(f.Content()),
		UpdatedAt: f.UpdatedAt().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *FileHandler) GetBinaryFileRaw(w http.ResponseWriter, r *http.Request) {
	f, ok := h.authorizeBinaryFileAccess(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+f.Name())
	_, _ = w.Write(f.Content())
}

func (h *FileHandler) ApplyFileChanges(w http.ResponseWriter, r *http.Request) {
	f, ok := h.authorizeTypstFileAccess(w, r)
	if !ok {
		return
	}

	var jsonReq jsonApplyFileChangesRequest
	if errJSON := json.NewDecoder(r.Body).Decode(&jsonReq); errJSON != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid request body"))
		return
	}

	req := fileApp.ApplyFileChangesRequest{
		FileID: f.ID(),
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
	projectID, ok := h.authorizeProjectAccess(w, r)
	if !ok {
		return
	}

	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid file ID"))
		return
	}

	var fileProjectID uuid.UUID
	if tf, errT := h.fileService.GetTypstFile(r.Context(), fileID); errT == nil {
		fileProjectID = tf.ProjectID()
	} else if bf, errB := h.fileService.GetBinaryFile(r.Context(), fileID); errB == nil {
		fileProjectID = bf.ProjectID()
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("File not found"))
		return
	}

	if fileProjectID != projectID {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("File does not belong to the specified project"))
		return
	}

	if errDelete := h.fileService.DeleteFile(r.Context(), fileID); errDelete != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Failed to delete file: " + errDelete.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
