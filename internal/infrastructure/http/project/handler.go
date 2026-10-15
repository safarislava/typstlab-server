package project

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	domain "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

type ProjectService interface {
	Create(ctx context.Context, req projectApp.CreateRequest) (*projectApp.CreateResponse, error)
	AddMember(ctx context.Context, projectID uuid.UUID, email string) (*projectApp.MemberDTO, error)
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]projectApp.MemberDTO, error)
}

type Handler struct {
	projectService ProjectService
}

func NewHandler(projectService ProjectService) *Handler {
	return &Handler{
		projectService: projectService,
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

	resp, err := h.projectService.Create(r.Context(), req)
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

type JSONMemberResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type jsonAddMemberRequest struct {
	Email string `json:"email"`
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	members, err := h.projectService.ListMembers(r.Context(), p.ID())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	resp := make([]JSONMemberResponse, len(members))
	for i, m := range members {
		resp[i] = JSONMemberResponse{
			ID:    m.ID.String(),
			Email: m.Email,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	var req jsonAddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid json body"))
		return
	}

	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("email is required"))
		return
	}

	member, err := h.projectService.AddMember(r.Context(), p.ID(), req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyMember) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("user is already a member of the project"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	resp := JSONMemberResponse{
		ID:    member.ID.String(),
		Email: member.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid user id"))
		return
	}

	if err := h.projectService.RemoveMember(r.Context(), p.ID(), userID); err != nil {
		if errors.Is(err, domain.ErrNoUsers) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("cannot remove the last member of the project"))
			return
		}
		if errors.Is(err, domain.ErrUserNotInProject) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("user is not a member of the project"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
