package di

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	appAuth "github.com/safarislava/typstlab-server/internal/application/auth"
	entryApp "github.com/safarislava/typstlab-server/internal/application/entry"
	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	binaryFile "github.com/safarislava/typstlab-server/internal/application/file/binary"
	typstFile "github.com/safarislava/typstlab-server/internal/application/file/typst"
	metadataApp "github.com/safarislava/typstlab-server/internal/application/metadata"
	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	syncFile "github.com/safarislava/typstlab-server/internal/application/sync/file"
	syncMetadata "github.com/safarislava/typstlab-server/internal/application/sync/metadata"
	userApp "github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/domain/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/auth"
	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/crdt"
	authHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http/auth"
	fileHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http/file"
	middlewareHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
	projectHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http/project"
	syncHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http/sync"
	userHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http/user"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/composite"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/memory"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/postgres"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/s3"
)

const headerContentType = "Content-Type"

// FileRepository groups all file-related repository operations across typst, binary, and metadata.
type FileRepository interface {
	fileApp.Repository
	typstFile.Repository
	binaryFile.Repository
	metadataApp.Repository
}

// Option configures DI container initialization.
type Option func(*Container)

// WithPool sets a custom PostgreSQL connection pool.
func WithPool(pool *pgxpool.Pool) Option {
	return func(c *Container) {
		c.pool = pool
	}
}

// WithS3Storage sets a custom S3 storage client.
func WithS3Storage(storage s3.Storage) Option {
	return func(c *Container) {
		c.s3Storage = storage
	}
}

// WithMemoryRepositories forces the container to use in-memory repositories.
func WithMemoryRepositories() Option {
	return func(c *Container) {
		c.useMemory = true
	}
}

// Container holds dependencies and lazily initializes them on demand.
type Container struct {
	cfg       *config.Config
	useMemory bool

	// Infrastructure / Persistence
	pool          *pgxpool.Pool
	poolOnce      sync.Once
	s3Storage     s3.Storage
	s3StorageOnce sync.Once

	// Repositories
	projectRepo     projectApp.Repository
	projectRepoOnce sync.Once

	fileRepo     FileRepository
	fileRepoOnce sync.Once

	userRepo     userApp.Repository
	userRepoOnce sync.Once

	sessionRepo     sessionApp.Repository
	sessionRepoOnce sync.Once

	// Infrastructure
	hasher     *auth.BcryptHasher
	hasherOnce sync.Once

	tokenService     *auth.JWTTokenService
	tokenServiceOnce sync.Once

	yjsMerger     *crdt.YjsMerger
	yjsMergerOnce sync.Once

	// Application Services
	entryService     *entryApp.Service
	entryServiceOnce sync.Once

	projectService     *projectApp.Service
	projectServiceOnce sync.Once

	typstFileService     *typstFile.Service
	typstFileServiceOnce sync.Once

	binaryFileService     *binaryFile.Service
	binaryFileServiceOnce sync.Once

	fileService     *fileApp.Service
	fileServiceOnce sync.Once

	metadataService     *metadataApp.Service
	metadataServiceOnce sync.Once

	syncMetadataService     *syncMetadata.Service
	syncMetadataServiceOnce sync.Once

	syncFileService     *syncFile.Service
	syncFileServiceOnce sync.Once

	syncService     *syncApp.Service
	syncServiceOnce sync.Once

	userService     *userApp.Service
	userServiceOnce sync.Once

	sessionService     *sessionApp.Service
	sessionServiceOnce sync.Once

	authService     *appAuth.Service
	authServiceOnce sync.Once

	// HTTP Handlers & Middlewares
	projectHandler     *projectHttp.Handler
	projectHandlerOnce sync.Once

	fileHandler     *fileHttp.Handler
	fileHandlerOnce sync.Once

	syncHandler     *syncHttp.Handler
	syncHandlerOnce sync.Once

	userHandler     *userHttp.Handler
	userHandlerOnce sync.Once

	authHandler     *authHttp.Handler
	authHandlerOnce sync.Once

	authMiddleware     *middlewareHttp.AuthMiddleware
	authMiddlewareOnce sync.Once

	accessMiddleware     *middlewareHttp.AccessMiddleware
	accessMiddlewareOnce sync.Once

	// Router
	router     *chi.Mux
	routerOnce sync.Once
}

// New creates a new lazy DI Container instance.
func New(cfg *config.Config, opts ...Option) *Container {
	c := &Container{
		cfg: cfg,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Config returns the application configuration.
func (c *Container) Config() *config.Config {
	return c.cfg
}

// Close closes any long-lived resources such as the database pool.
func (c *Container) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

// Pool lazily initializes and returns the PostgreSQL connection pool.
func (c *Container) Pool() *pgxpool.Pool {
	c.poolOnce.Do(func() {
		if c.pool == nil && !c.useMemory && c.cfg != nil && c.cfg.DatabaseURL != "" {
			pool, err := postgres.NewPool(context.Background(), c.cfg.DatabaseURL)
			if err == nil {
				c.pool = pool
			}
		}
	})
	return c.pool
}

// S3Storage lazily initializes and returns the S3 blob storage client.
func (c *Container) S3Storage() s3.Storage {
	c.s3StorageOnce.Do(func() {
		if c.s3Storage == nil && !c.useMemory && c.cfg != nil && c.cfg.S3.Endpoint != "" {
			client, err := s3.NewClient(&s3.Config{
				Endpoint:  c.cfg.S3.Endpoint,
				Bucket:    c.cfg.S3.Bucket,
				AccessKey: c.cfg.S3.AccessKey,
				SecretKey: c.cfg.S3.SecretKey,
				UseSSL:    c.cfg.S3.UseSSL,
				Region:    c.cfg.S3.Region,
			})
			if err == nil {
				c.s3Storage = client
			}
		}
	})
	return c.s3Storage
}

// ProjectRepo lazily initializes and returns the project repository.
func (c *Container) ProjectRepo() projectApp.Repository {
	c.projectRepoOnce.Do(func() {
		if c.projectRepo == nil {
			c.projectRepo = c.buildProjectRepo()
		}
	})
	return c.projectRepo
}

func (c *Container) buildProjectRepo() projectApp.Repository {
	if !c.useMemory {
		if pool := c.Pool(); pool != nil {
			return postgres.NewProjectRepository(pool)
		}
	}
	return memory.NewMemoryProjectRepository()
}

// FileRepo lazily initializes and returns the composite or in-memory file repository.
func (c *Container) FileRepo() FileRepository {
	c.fileRepoOnce.Do(func() {
		if c.fileRepo == nil {
			c.fileRepo = c.buildFileRepo()
		}
	})
	return c.fileRepo
}

func (c *Container) buildFileRepo() FileRepository {
	if c.useMemory {
		return memory.NewMemoryFileRepository()
	}
	pool := c.Pool()
	if pool == nil {
		return memory.NewMemoryFileRepository()
	}
	pgRepo := postgres.NewFileRepository(pool)
	s3Storage := c.S3Storage()
	if s3Storage == nil {
		return pgRepo
	}
	compRepo, err := composite.NewFileRepository(pgRepo, s3Storage)
	if err != nil {
		return pgRepo
	}
	return compRepo
}

// UserRepo lazily initializes and returns the user repository.
func (c *Container) UserRepo() userApp.Repository {
	c.userRepoOnce.Do(func() {
		if c.userRepo == nil {
			c.userRepo = c.buildUserRepo()
		}
	})
	return c.userRepo
}

func (c *Container) buildUserRepo() userApp.Repository {
	if !c.useMemory {
		if pool := c.Pool(); pool != nil {
			return postgres.NewUserRepository(pool)
		}
	}
	return memory.NewMemoryUserRepository()
}

// SessionRepo lazily initializes and returns the session repository.
func (c *Container) SessionRepo() sessionApp.Repository {
	c.sessionRepoOnce.Do(func() {
		if c.sessionRepo == nil {
			c.sessionRepo = c.buildSessionRepo()
		}
	})
	return c.sessionRepo
}

func (c *Container) buildSessionRepo() sessionApp.Repository {
	if !c.useMemory {
		if pool := c.Pool(); pool != nil {
			return postgres.NewSessionRepository(pool)
		}
	}
	return memory.NewMemorySessionRepository()
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

// EntryService lazily initializes and returns the entry application service.
func (c *Container) EntryService() *entryApp.Service {
	c.entryServiceOnce.Do(func() {
		c.entryService = entryApp.NewService()
	})
	return c.entryService
}

// ProjectService lazily initializes and returns the project application service.
func (c *Container) ProjectService() *projectApp.Service {
	c.projectServiceOnce.Do(func() {
		c.projectService = projectApp.NewService(c.ProjectRepo())
	})
	return c.projectService
}

// TypstFileService lazily initializes and returns the typst file application service.
func (c *Container) TypstFileService() *typstFile.Service {
	c.typstFileServiceOnce.Do(func() {
		c.typstFileService = typstFile.NewService(c.FileRepo())
	})
	return c.typstFileService
}

// BinaryFileService lazily initializes and returns the binary file application service.
func (c *Container) BinaryFileService() *binaryFile.Service {
	c.binaryFileServiceOnce.Do(func() {
		c.binaryFileService = binaryFile.NewService(c.FileRepo())
	})
	return c.binaryFileService
}

// FileService lazily initializes and returns the file application service.
func (c *Container) FileService() *fileApp.Service {
	c.fileServiceOnce.Do(func() {
		c.fileService = fileApp.NewService(c.FileRepo(), c.TypstFileService(), c.BinaryFileService())
	})
	return c.fileService
}

// MetadataService lazily initializes and returns the metadata application service.
func (c *Container) MetadataService() *metadataApp.Service {
	c.metadataServiceOnce.Do(func() {
		c.metadataService = metadataApp.NewService(c.FileRepo())
	})
	return c.metadataService
}

// SyncMetadataService lazily initializes and returns the metadata sync service.
func (c *Container) SyncMetadataService() *syncMetadata.Service {
	c.syncMetadataServiceOnce.Do(func() {
		c.syncMetadataService = syncMetadata.NewService(c.MetadataService(), c.YjsMerger())
	})
	return c.syncMetadataService
}

// SyncFileService lazily initializes and returns the file sync service.
func (c *Container) SyncFileService() *syncFile.Service {
	c.syncFileServiceOnce.Do(func() {
		c.syncFileService = syncFile.NewService(
			c.FileService(),
			c.TypstFileService(),
			c.YjsMerger(),
			c.YjsMerger(),
		)
	})
	return c.syncFileService
}

// SyncService lazily initializes and returns the sync application service.
func (c *Container) SyncService() *syncApp.Service {
	c.syncServiceOnce.Do(func() {
		c.syncService = syncApp.NewService(
			c.SyncMetadataService(),
			c.SyncFileService(),
		)
	})
	return c.syncService
}

// UserService lazily initializes and returns the user application service.
func (c *Container) UserService() *userApp.Service {
	c.userServiceOnce.Do(func() {
		c.userService = userApp.NewService(c.UserRepo(), c.Hasher())
	})
	return c.userService
}

// SessionService lazily initializes and returns the session application service.
func (c *Container) SessionService() *sessionApp.Service {
	c.sessionServiceOnce.Do(func() {
		c.sessionService = sessionApp.NewService(c.SessionRepo())
	})
	return c.sessionService
}

// AuthService lazily initializes and returns the auth application service.
func (c *Container) AuthService() *appAuth.Service {
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
func (c *Container) ProjectHandler() *projectHttp.Handler {
	c.projectHandlerOnce.Do(func() {
		c.projectHandler = projectHttp.NewHandler(c.ProjectService())
	})
	return c.projectHandler
}

// FileHandler lazily initializes and returns the file HTTP handler.
func (c *Container) FileHandler() *fileHttp.Handler {
	c.fileHandlerOnce.Do(func() {
		c.fileHandler = fileHttp.NewHandler(
			c.TypstFileService(),
			c.BinaryFileService(),
			c.FileService(),
			c.SyncService(),
		)
	})
	return c.fileHandler
}

// SyncHandler lazily initializes and returns the sync HTTP handler.
func (c *Container) SyncHandler() *syncHttp.Handler {
	c.syncHandlerOnce.Do(func() {
		c.syncHandler = syncHttp.NewHandler(c.SyncService())
	})
	return c.syncHandler
}

// UserHandler lazily initializes and returns the user HTTP handler.
func (c *Container) UserHandler() *userHttp.Handler {
	c.userHandlerOnce.Do(func() {
		c.userHandler = userHttp.NewHandler(c.UserService())
	})
	return c.userHandler
}

// AuthHandler lazily initializes and returns the auth HTTP handler.
func (c *Container) AuthHandler() *authHttp.Handler {
	c.authHandlerOnce.Do(func() {
		c.authHandler = authHttp.NewHandler(c.AuthService())
	})
	return c.authHandler
}

// AuthMiddleware lazily initializes and returns the auth middleware.
func (c *Container) AuthMiddleware() *middlewareHttp.AuthMiddleware {
	c.authMiddlewareOnce.Do(func() {
		c.authMiddleware = middlewareHttp.NewAuthMiddleware(c.AuthService())
	})
	return c.authMiddleware
}

// AccessMiddleware lazily initializes and returns the access middleware.
func (c *Container) AccessMiddleware() *middlewareHttp.AccessMiddleware {
	c.accessMiddlewareOnce.Do(func() {
		c.accessMiddleware = middlewareHttp.NewAccessMiddleware(
			c.ProjectService(),
			c.TypstFileService(),
			c.BinaryFileService(),
		)
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
		r.Use(middlewareHttp.RequireAuthenticated)
		r.Use(middlewareHttp.RequireRole(user.RoleUser))

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
