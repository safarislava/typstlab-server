package http

import (
	"encoding/json"
	"net/http"

	"github.com/safarislava/typstlab-server/internal/application/user"
)

type UserHandler struct {
	service *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

type jsonRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type JSONRegisterResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type jsonLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type JSONLoginResponse struct {
	Token string `json:"token"`
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

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	jsonResp := JSONRegisterResponse{
		ID:    resp.ID.String(),
		Email: resp.Email,
		Role:  string(resp.Role),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(jsonResp)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var jsonReq jsonLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := user.LoginRequest{
		Email:    jsonReq.Email,
		Password: jsonReq.Password,
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	jsonResp := JSONLoginResponse{
		Token: resp.Token.Value(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(jsonResp)
}
