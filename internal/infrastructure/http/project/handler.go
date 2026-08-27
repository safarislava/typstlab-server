package project

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

type Service interface {
	Create(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

type jsonCreateRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type JSONCreateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
}

func newJSONCreateResponse(response *projectApp.CreateResponse) JSONCreateResponse {
	return JSONCreateResponse{
		ID:        response.ID.String(),
		Name:      response.Name,
		UpdatedAt: response.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var jsonReq jsonCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(jsonReq.ID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid or missing project id: " + err.Error()))
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	req := projectApp.CreateRequest{
		ID:     projectID,
		UserID: userID,
		Name:   jsonReq.Name,
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	jsonResp := newJSONCreateResponse(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(jsonResp)
}

type JSONProjectResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	UserIDs   []string `json:"user_ids"`
	UpdatedAt string   `json:"updated_at"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	userIDs := make([]string, len(p.UserIDs()))
	for i, id := range p.UserIDs() {
		userIDs[i] = id.String()
	}

	resp := JSONProjectResponse{
		ID:        p.ID().String(),
		Name:      p.Name(),
		UserIDs:   userIDs,
		UpdatedAt: p.UpdatedAt().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
