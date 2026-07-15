package http

import (
	"encoding/json"
	"net/http"
	"time"

	application "github.com/safarislava/typstlab-server/internal/application/project"
)

type ProjectHandler struct {
	service *application.Service
}

func NewProjectHandler(service *application.Service) *ProjectHandler {
	return &ProjectHandler{
		service: service,
	}
}

type jsonCreateProjectRequest struct {
	Name string `json:"name"`
}

type JSONCreateProjectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
}

func NewJSONCreateProjectResponse(response *application.CreateProjectResponse) JSONCreateProjectResponse {
	return JSONCreateProjectResponse{
		ID:        response.ID.String(),
		Name:      response.Name,
		UpdatedAt: response.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var jsonRequest jsonCreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	request := application.CreateProjectRequest{
		Name: jsonRequest.Name,
	}

	response, err := h.service.CreateProject(r.Context(), request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	jsonResponse := NewJSONCreateProjectResponse(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(jsonResponse)
}
