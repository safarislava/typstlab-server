package invite

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domainInvite "github.com/safarislava/typstlab-server/internal/domain/invite"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

type Service interface {
	CreateInvite(ctx context.Context, projectID, createdBy uuid.UUID, expiresAt time.Time) (*domainInvite.Invite, error)
	ListInvites(ctx context.Context, projectID uuid.UUID) ([]*domainInvite.Invite, error)
	RevokeInvite(ctx context.Context, projectID, inviteID uuid.UUID) error
	AcceptInvite(ctx context.Context, code string, userID uuid.UUID) (*domainProject.Project, error)
}

type Handler struct {
	inviteService Service
}

func NewHandler(inviteService Service) *Handler {
	return &Handler{
		inviteService: inviteService,
	}
}

type jsonCreateInviteRequest struct {
	ExpiresAt *time.Time `json:"expires_at"`
}

type JSONInviteResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Code      string `json:"code"`
	CreatedBy string `json:"created_by"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

func newJSONInviteResponse(inv *domainInvite.Invite) JSONInviteResponse {
	return JSONInviteResponse{
		ID:        inv.ID().String(),
		ProjectID: inv.ProjectID().String(),
		Code:      inv.Code(),
		CreatedBy: inv.CreatedBy().String(),
		ExpiresAt: inv.ExpiresAt().Format(time.RFC3339),
		CreatedAt: inv.CreatedAt().Format(time.RFC3339),
	}
}

type JSONProjectResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	UserIDs   []string `json:"user_ids"`
	UpdatedAt string   `json:"updated_at"`
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	var expiresAt time.Time
	if r.ContentLength > 0 {
		var req jsonCreateInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.ExpiresAt != nil {
			expiresAt = *req.ExpiresAt
		}
	}

	inv, err := h.inviteService.CreateInvite(r.Context(), p.ID(), userID, expiresAt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	resp := newJSONInviteResponse(inv)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	invites, err := h.inviteService.ListInvites(r.Context(), p.ID())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	resp := make([]JSONInviteResponse, len(invites))
	for i, inv := range invites {
		resp[i] = newJSONInviteResponse(inv)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Project not found in context"))
		return
	}

	inviteIDStr := chi.URLParam(r, "inviteID")
	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid invite id"))
		return
	}

	if err := h.inviteService.RevokeInvite(r.Context(), p.ID(), inviteID); err != nil {
		if errors.Is(err, domainInvite.ErrInviteNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("invite not found"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invite code is required"))
		return
	}

	p, err := h.inviteService.AcceptInvite(r.Context(), code, userID)
	if err != nil {
		if errors.Is(err, domainInvite.ErrInviteNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("invite not found"))
			return
		}
		if errors.Is(err, domainInvite.ErrInviteExpired) {
			w.WriteHeader(http.StatusGone)
			_, _ = w.Write([]byte("invite has expired"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
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
