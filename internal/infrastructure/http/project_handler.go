package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	application "github.com/safarislava/typstlab-server/internal/application/project"
)

type ProjectHandler struct {
	service application.UseCase
}

func NewProjectHandler(service application.UseCase) *ProjectHandler {
	return &ProjectHandler{
		service: service,
	}
}

type jsonCreateRequest struct {
	Name string `json:"name"`
}

type JSONCreateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
}

func newJSONCreateResponse(response *application.CreateResponse) JSONCreateResponse {
	return JSONCreateResponse{
		ID:        response.ID.String(),
		Name:      response.Name,
		UpdatedAt: response.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var jsonReq jsonCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	req := application.CreateRequest{
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

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid project ID"))
		return
	}

	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	p, err := h.service.Get(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Project not found"))
		return
	}

	if !p.HasUser(userID) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden"))
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
