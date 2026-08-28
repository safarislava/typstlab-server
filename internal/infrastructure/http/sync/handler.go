package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

type Service interface {
	Sync(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error)
}

type Handler struct {
	syncService Service
}

func NewHandler(syncService Service) *Handler {
	return &Handler{
		syncService: syncService,
	}
}

type JSONSyncRequest struct {
	MetadataDelta       []byte            `json:"metadata_delta,omitempty"`
	MetadataStateVector []byte            `json:"metadata_state_vector,omitempty"`
	ContentVectors      map[string][]byte `json:"content_vectors,omitempty"`
}

type JSONInstructionResponse struct {
	Action string `json:"action"`
	FileID string `json:"file_id"`
	Delta  []byte `json:"delta,omitempty"`
}

type JSONSyncResponse struct {
	MetadataDelta []byte                    `json:"metadata_delta,omitempty"`
	Instructions  []JSONInstructionResponse `json:"instructions"`
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
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

	contentVectors, err := parseContentVectors(jsonReq.ContentVectors)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	appReq := &syncApp.Request{
		MetadataDelta:       jsonReq.MetadataDelta,
		MetadataStateVector: jsonReq.MetadataStateVector,
		ContentVectors:      contentVectors,
	}

	resp, err := h.syncService.Sync(r.Context(), p.ID(), appReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	h.writeResponse(w, resp)
}

func parseContentVectors(raw map[string][]byte) (map[uuid.UUID][]byte, error) {
	if raw == nil {
		return nil, nil
	}
	res := make(map[uuid.UUID][]byte, len(raw))
	for k, v := range raw {
		id, err := uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("invalid file id in content vectors: %s", k)
		}
		res[id] = v
	}
	return res, nil
}

func (h *Handler) writeResponse(w http.ResponseWriter, resp *syncApp.Response) {
	instructions := make([]JSONInstructionResponse, 0, len(resp.Instructions))
	for _, instruction := range resp.Instructions {
		instructions = append(instructions, JSONInstructionResponse{
			Action: string(instruction.Action),
			FileID: instruction.FileID.String(),
			Delta:  instruction.Delta,
		})
	}

	jsonResp := JSONSyncResponse{
		MetadataDelta: resp.MetadataDelta,
		Instructions:  instructions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(jsonResp)
}
