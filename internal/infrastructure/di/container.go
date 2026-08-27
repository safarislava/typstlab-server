package di

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	appAuth "github.com/safarislava/typstlab-server/internal/application/auth"
	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	userApp "github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/domain/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/auth"
	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/crdt"
	projectHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence"
)

const headerContentType = "Content-Type"

// Container holds dependencies and lazily initializes them on demand.
type Container struct {
	cfg *config.Config

	// Repositories
	projectRepo     *persistence.MemoryProjectRepository
	projectRepoOnce sync.Once

	fileRepo     *persistence.MemoryFileRepository
	fileRepoOnce sync.Once

	userRepo     *persistence.MemoryUserRepository
	userRepoOnce sync.Once

	sessionRepo     *persistence.MemorySessionRepository
	sessionRepoOnce sync.Once

	// Infrastructure
	hasher     *auth.BcryptHasher
	hasherOnce sync.Once

	tokenService     *auth.JWTTokenService
	tokenServiceOnce sync.Once

	yjsMerger     *crdt.YjsMerger
	yjsMergerOnce sync.Once

	// Application Services / UseCases
	projectService     projectApp.UseCase
	projectServiceOnce sync.Once

	fileService     fileApp.UseCase
	fileServiceOnce sync.Once

	syncService     syncApp.UseCase
	syncServiceOnce sync.Once

	userService     userApp.UseCase
	userServiceOnce sync.Once

	sessionService     sessionApp.UseCase
	sessionServiceOnce sync.Once

	authService     appAuth.UseCase
	authServiceOnce sync.Once

	// HTTP Handlers & Middlewares
	projectHandler     *projectHttp.ProjectHandler
	projectHandlerOnce sync.Once

	fileHandler     *projectHttp.FileHandler
	fileHandlerOnce sync.Once

	syncHandler     *projectHttp.SyncHandler
	syncHandlerOnce sync.Once

	userHandler     *projectHttp.UserHandler
	userHandlerOnce sync.Once

	authHandler     *projectHttp.AuthHandler
	authHandlerOnce sync.Once

	authMiddleware     *projectHttp.AuthMiddleware
	authMiddlewareOnce sync.Once

	accessMiddleware     *projectHttp.AccessMiddleware
	accessMiddlewareOnce sync.Once

	// Router
	router     *chi.Mux
	routerOnce sync.Once
}

// New creates a new lazy DI Container instance.
func New(cfg *config.Config) *Container {
	return &Container{
		cfg: cfg,
	}
}

// Config returns the application configuration.
func (c *Container) Config() *config.Config {
	return c.cfg
}

// ProjectRepo lazily initializes and returns the project repository.
func (c *Container) ProjectRepo() *persistence.MemoryProjectRepository {
	c.projectRepoOnce.Do(func() {
		c.projectRepo = persistence.NewMemoryProjectRepository()
	})
	return c.projectRepo
}

// FileRepo lazily initializes and returns the file repository.
func (c *Container) FileRepo() *persistence.MemoryFileRepository {
	c.fileRepoOnce.Do(func() {
		c.fileRepo = persistence.NewMemoryFileRepository()
	})
	return c.fileRepo
}

// UserRepo lazily initializes and returns the user repository.
func (c *Container) UserRepo() *persistence.MemoryUserRepository {
	c.userRepoOnce.Do(func() {
		c.userRepo = persistence.NewMemoryUserRepository()
	})
	return c.userRepo
}

// SessionRepo lazily initializes and returns the session repository.
func (c *Container) SessionRepo() *persistence.MemorySessionRepository {
	c.sessionRepoOnce.Do(func() {
		c.sessionRepo = persistence.NewMemorySessionRepository()
	})
	return c.sessionRepo
}

// Hasher lazily initializes and returns the password hasher.
func (c *Container) Hasher() *auth.BcryptHasher {
	c.hasherOnce.Do(func() {
		c.hasher = auth.NewBcryptHasher(0)
	})
	return c.hasher
}

// TokenService lazily initializes and returns the JWT token service.
func (c *Container) TokenService() *auth.JWTTokenService {
	c.tokenServiceOnce.Do(func() {
		c.tokenService = auth.NewJWTTokenService(c.cfg.JWTSecret, 24*time.Hour)
	})
	return c.tokenService
}

// YjsMerger lazily initializes and returns the Yjs CRDT merger.
func (c *Container) YjsMerger() *crdt.YjsMerger {
	c.yjsMergerOnce.Do(func() {
		c.yjsMerger = crdt.NewYjsMerger()
	})
	return c.yjsMerger
}

// ProjectService lazily initializes and returns the project application service.
func (c *Container) ProjectService() projectApp.UseCase {
	c.projectServiceOnce.Do(func() {
		c.projectService = projectApp.NewService(c.ProjectRepo())
	})
	return c.projectService
}

// FileService lazily initializes and returns the file application service.
func (c *Container) FileService() fileApp.UseCase {
	c.fileServiceOnce.Do(func() {
		c.fileService = fileApp.NewService(c.FileRepo(), c.YjsMerger())
	})
	return c.fileService
}

// SyncService lazily initializes and returns the sync application service.
func (c *Container) SyncService() syncApp.UseCase {
	c.syncServiceOnce.Do(func() {
		c.syncService = syncApp.NewService(c.FileRepo())
	})
	return c.syncService
}

// UserService lazily initializes and returns the user application service.
func (c *Container) UserService() userApp.UseCase {
	c.userServiceOnce.Do(func() {
		c.userService = userApp.NewService(c.UserRepo(), c.Hasher())
	})
	return c.userService
}

// SessionService lazily initializes and returns the session application service.
func (c *Container) SessionService() sessionApp.UseCase {
	c.sessionServiceOnce.Do(func() {
		c.sessionService = sessionApp.NewService(c.SessionRepo())
	})
	return c.sessionService
}

// AuthService lazily initializes and returns the auth application service.
func (c *Container) AuthService() appAuth.UseCase {
	c.authServiceOnce.Do(func() {
		c.authService = appAuth.NewService(
			c.UserService(),
			c.SessionService(),
			c.TokenService(),
			c.Hasher(),
		)
	})
	return c.authService
}

// ProjectHandler lazily initializes and returns the project HTTP handler.
func (c *Container) ProjectHandler() *projectHttp.ProjectHandler {
	c.projectHandlerOnce.Do(func() {
		c.projectHandler = projectHttp.NewProjectHandler(c.ProjectService())
	})
	return c.projectHandler
}

// FileHandler lazily initializes and returns the file HTTP handler.
func (c *Container) FileHandler() *projectHttp.FileHandler {
	c.fileHandlerOnce.Do(func() {
		c.fileHandler = projectHttp.NewFileHandler(c.FileService())
	})
	return c.fileHandler
}

// SyncHandler lazily initializes and returns the sync HTTP handler.
func (c *Container) SyncHandler() *projectHttp.SyncHandler {
	c.syncHandlerOnce.Do(func() {
		c.syncHandler = projectHttp.NewSyncHandler(c.SyncService())
	})
	return c.syncHandler
}

// UserHandler lazily initializes and returns the user HTTP handler.
func (c *Container) UserHandler() *projectHttp.UserHandler {
	c.userHandlerOnce.Do(func() {
		c.userHandler = projectHttp.NewUserHandler(c.UserService())
	})
	return c.userHandler
}

// AuthHandler lazily initializes and returns the auth HTTP handler.
func (c *Container) AuthHandler() *projectHttp.AuthHandler {
	c.authHandlerOnce.Do(func() {
		c.authHandler = projectHttp.NewAuthHandler(c.AuthService())
	})
	return c.authHandler
}

// AuthMiddleware lazily initializes and returns the auth middleware.
func (c *Container) AuthMiddleware() *projectHttp.AuthMiddleware {
	c.authMiddlewareOnce.Do(func() {
		c.authMiddleware = projectHttp.NewAuthMiddleware(c.AuthService())
	})
	return c.authMiddleware
}

// AccessMiddleware lazily initializes and returns the access middleware.
func (c *Container) AccessMiddleware() *projectHttp.AccessMiddleware {
	c.accessMiddlewareOnce.Do(func() {
		c.accessMiddleware = projectHttp.NewAccessMiddleware(c.ProjectService(), c.FileService())
	})
	return c.accessMiddleware
}

// Router lazily initializes the Chi router with middlewares and registered routes.
func (c *Container) Router() *chi.Mux {
	c.routerOnce.Do(func() {
		r := chi.NewRouter()
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   c.cfg.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", headerContentType, "X-CSRF-Token", "X-Requested-With"},
			ExposedHeaders:   []string{"Link", "Content-Length", headerContentType},
			AllowCredentials: true,
			MaxAge:           300,
		}))

		c.registerRoutes(r)
		c.router = r
	})
	return c.router
}

func (c *Container) registerRoutes(r *chi.Mux) {
	userHandler := c.UserHandler()
	authHandler := c.AuthHandler()
	projectHandler := c.ProjectHandler()
	fileHandler := c.FileHandler()
	syncHandler := c.SyncHandler()
	authMiddleware := c.AuthMiddleware()
	accessMiddleware := c.AccessMiddleware()

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
			r.Route("/typst/{fileID}", func(r chi.Router) {
				r.Use(accessMiddleware.FileAccess)
				r.Get("/", fileHandler.GetTypstFile)
				r.Post("/changes", fileHandler.ApplyFileChanges)
			})
			r.Route("/binary/{fileID}", func(r chi.Router) {
				r.Use(accessMiddleware.FileAccess)
				r.Get("/", fileHandler.GetBinaryFile)
				r.Get("/raw", fileHandler.GetBinaryFileRaw)
			})
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
