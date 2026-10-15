package middleware

import (
	"context"
	"net/http"
	"slices"

	"github.com/google/uuid"

	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	roleKey   contextKey = "role"
)

const AccessTokenCookieName = "access_token"

type AuthService interface {
	Authorize(t domainToken.Token) (uuid.UUID, domainUser.Role, error)
}

type AuthMiddleware struct {
	authService AuthService
}

func NewAuthMiddleware(authService AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func extractToken(r *http.Request) string {
	if cookie, err := r.Cookie(AccessTokenCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			next.ServeHTTP(w, r)
			return
		}

		t, err := domainToken.NewToken(tokenString)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		userID, role, err := m.authService.Authorize(t)
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
	return slices.Contains(allowedRoles, userRole)
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

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}
