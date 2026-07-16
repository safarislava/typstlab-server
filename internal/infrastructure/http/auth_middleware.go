package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/application/user"
	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	roleKey   contextKey = "role"
)

type AuthMiddleware struct {
	tokenService user.TokenService
}

func NewAuthMiddleware(tokenService user.TokenService) *AuthMiddleware {
	return &AuthMiddleware{
		tokenService: tokenService,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := parts[1]
		tok, err := domainToken.NewToken(tokenStr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		userID, role, err := m.tokenService.Validate(tok)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, roleKey, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
		if !ok || userID == uuid.Nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isRoleAllowed(userRole domainUser.Role, allowedRoles []domainUser.Role) bool {
	for _, role := range allowedRoles {
		if userRole == role {
			return true
		}
	}
	return false
}

func RequireRole(allowedRoles ...domainUser.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(roleKey).(domainUser.Role)
			if !ok || !isRoleAllowed(userRole, allowedRoles) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("Forbidden"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
