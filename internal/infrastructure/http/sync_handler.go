package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type SyncHandler struct {
	syncService syncApp.UseCase
}

func NewSyncHandler(syncService syncApp.UseCase) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

type JSONSyncFileRequest struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	YjsStateVector []byte `json:"yjs_state_vector,omitempty"`
}

type JSONSyncRequest struct {
	Files []JSONSyncFileRequest `json:"files"`
}

type JSONInstructionResponse struct {
	Action  string `json:"action"`
	FileID  string `json:"file_id"`
	NewName string `json:"new_name,omitempty"`
	Delta   []byte `json:"delta,omitempty"`
}

type JSONSyncResponse struct {
	Instructions []JSONInstructionResponse `json:"instructions"`
}

func (h *SyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
	p, ok := ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	var jsonReq JSONSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid request body"))
		return
	}

	files := make([]syncApp.FileRequest, 0, len(jsonReq.Files))
	for _, f := range jsonReq.Files {
		id, err := uuid.Parse(f.ID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid file id in sync request: " + f.ID))
			return
		}
		files = append(files, syncApp.FileRequest{
			ID:    id,
			Name:  f.Name,
			Type:  domainFile.Type(f.Type),
			State: f.YjsStateVector,
		})
	}

	appReq := &syncApp.Request{
		Files: files,
	}

	resp, err := h.syncService.Sync(r.Context(), p.ID(), appReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	instructions := make([]JSONInstructionResponse, 0, len(resp.Instructions))
	for _, instruction := range resp.Instructions {
		instructions = append(instructions, JSONInstructionResponse{
			Action:  string(instruction.Action),
			FileID:  instruction.FileID.String(),
			NewName: instruction.NewName,
			Delta:   instruction.Delta,
		})
	}

	jsonResp := JSONSyncResponse{
		Instructions: instructions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(jsonResp)
}
