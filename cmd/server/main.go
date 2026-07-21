package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/safarislava/typstlab-server/internal/domain/user"

	appAuth "github.com/safarislava/typstlab-server/internal/application/auth"
	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	userApp "github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/auth"
	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/crdt"
	projectHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http"
	projectDb "github.com/safarislava/typstlab-server/internal/infrastructure/persistence"
)

func main() {
	cfg := config.Load("configs/config.json")
	r := setupRouter(cfg)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

func setupRouter(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Projects components
	projectRepo := projectDb.NewMemoryProjectRepository()
	projectService := projectApp.NewService(projectRepo)
	projectHandler := projectHttp.NewProjectHandler(projectService)

	// Files components
	fileRepo := projectDb.NewMemoryFileRepository()
	yjsMerger := crdt.NewYjsMerger()
	fileService := fileApp.NewService(fileRepo, yjsMerger)
	fileHandler := projectHttp.NewFileHandler(fileService)

	// Sync components
	syncService := syncApp.NewService(fileRepo)
	syncHandler := projectHttp.NewSyncHandler(syncService)

	// Users / Auth components
	userRepo := projectDb.NewMemoryUserRepository()
	sessionRepo := projectDb.NewMemorySessionRepository()
	hasher := auth.NewBcryptHasher(0)
	tokenService := auth.NewJWTTokenService(cfg.JWTSecret, 24*time.Hour)
	userService := userApp.NewService(userRepo, hasher)
	sessionService := sessionApp.NewService(sessionRepo)
	authUseCase := appAuth.NewService(userService, sessionService, tokenService, hasher)
	userHandler := projectHttp.NewUserHandler(userService)
	authHandler := projectHttp.NewAuthHandler(authUseCase)
	authMiddleware := projectHttp.NewAuthMiddleware(authUseCase)
	accessMiddleware := projectHttp.NewAccessMiddleware(projectService, fileService)

	registerRoutes(r, userHandler, authHandler, projectHandler, fileHandler, syncHandler, authMiddleware, accessMiddleware)

	return r
}

func registerRoutes(
	r *chi.Mux,
	userHandler *projectHttp.UserHandler,
	authHandler *projectHttp.AuthHandler,
	projectHandler *projectHttp.ProjectHandler,
	fileHandler *projectHttp.FileHandler,
	syncHandler *projectHttp.SyncHandler,
	authMiddleware *projectHttp.AuthMiddleware,
	accessMiddleware *projectHttp.AccessMiddleware,
) {
	// Auth routes
	r.Post("/register", userHandler.Register)
	r.Post("/login", authHandler.Login)
	r.Post("/refresh", authHandler.Refresh)
	r.Post("/logout", authHandler.Logout)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(projectHttp.RequireAuthenticated)
		r.Use(projectHttp.RequireRole(user.RoleUser))

		r.Post("/projects", projectHandler.Create)

		r.Route("/projects/{projectID}", func(r chi.Router) {
			r.Use(accessMiddleware.ProjectAccess)
			r.Get("/", projectHandler.Get)
			r.Post("/files", fileHandler.UploadFile)
			r.Get("/files", fileHandler.ListProjectFiles)
			r.With(accessMiddleware.FileAccess).Delete("/files/{fileID}", fileHandler.DeleteFile)
			r.Post("/sync", syncHandler.Sync)
		})

		r.Route("/files", func(r chi.Router) {
			r.Use(accessMiddleware.FileAccess)
			r.Get("/typst/{fileID}", fileHandler.GetTypstFile)
			r.Post("/typst/{fileID}/changes", fileHandler.ApplyFileChanges)
			r.Get("/binary/{fileID}", fileHandler.GetBinaryFile)
			r.Get("/binary/{fileID}/raw", fileHandler.GetBinaryFileRaw)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
