package http

import (
	"encoding/json"
	"net/http"

	"github.com/safarislava/typstlab-server/internal/application/auth"
	tokenDomain "github.com/safarislava/typstlab-server/internal/domain/token"
)

const refreshTokenCookieName = "refresh_token"

type AuthHandler struct {
	authService auth.UseCase
}

func NewAuthHandler(authService auth.UseCase) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type jsonLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type JSONLoginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var jsonReq jsonLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := auth.LoginRequest{
		Email:    jsonReq.Email,
		Password: jsonReq.Password,
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.setSessionCookie(w, resp.RefreshToken.Token().Value(), 30*24*3600)
	h.writeJSON(w, http.StatusOK, JSONLoginResponse{Token: resp.AccessToken.Value()})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Missing refresh token cookie")
		return
	}

	rt, err := tokenDomain.NewToken(cookie.Value)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "Invalid refresh token format")
		return
	}

	resp, err := h.authService.Refresh(r.Context(), auth.RefreshRequest{
		RefreshToken: rt,
	})
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.setSessionCookie(w, resp.RefreshToken.Token().Value(), 30*24*3600)
	h.writeJSON(w, http.StatusOK, JSONLoginResponse{Token: resp.AccessToken.Value()})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err == nil {
		if rt, err := tokenDomain.NewToken(cookie.Value); err == nil {
			_ = h.authService.Logout(r.Context(), rt)
		}
	}

	h.setSessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

func (h *AuthHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}
