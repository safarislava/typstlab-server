package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/safarislava/typstlab-server/internal/domain/user"

	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	userApp "github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/auth"
	projectHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http"
	projectDb "github.com/safarislava/typstlab-server/internal/infrastructure/persistence"
)

func main() {
	r := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(":"+port, r))
}

func setupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Projects components
	projectRepo := projectDb.NewMemoryProjectRepository()
	projectService := projectApp.NewService(projectRepo)
	projectHandler := projectHttp.NewProjectHandler(projectService)

	// Users / Auth components
	userRepo := projectDb.NewMemoryUserRepository()
	hasher := auth.NewBcryptHasher(0)
	tokenService := auth.NewMemoryTokenService()
	userService := userApp.NewService(userRepo, hasher, tokenService)
	userHandler := projectHttp.NewUserHandler(userService)
	authMiddleware := projectHttp.NewAuthMiddleware(tokenService)

	// Auth routes
	r.Post("/register", userHandler.Register)
	r.Post("/login", userHandler.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(projectHttp.RequireAuthenticated)
		r.Use(projectHttp.RequireRole(user.RoleUser))

		r.Post("/projects", projectHandler.Create)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}
