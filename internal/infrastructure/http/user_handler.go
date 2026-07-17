package http

import (
	"encoding/json"
	"net/http"

	"github.com/safarislava/typstlab-server/internal/application/user"
)

type UserHandler struct {
	userService *user.Service
}

func NewUserHandler(userService *user.Service) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type jsonRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type jsonRegisterResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var jsonReq jsonRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := user.RegisterRequest{
		Email:    jsonReq.Email,
		Password: jsonReq.Password,
		Role:     jsonReq.Role,
	}

	resp, err := h.userService.Register(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	jsonResp := jsonRegisterResponse{
		ID:    resp.ID.String(),
		Email: resp.Email,
		Role:  string(resp.Role),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(jsonResp)
}
