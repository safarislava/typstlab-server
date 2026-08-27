package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
)

const (
	projectContextKey contextKey = "project"
	fileContextKey    contextKey = "file"
)

type ProjectService interface {
	Get(ctx context.Context, projectID uuid.UUID) (*domainProject.Project, error)
}

type FileService interface {
	GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	GetBinaryFile(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error)
}

type AccessMiddleware struct {
	projectService ProjectService
	fileService    FileService
}

func NewAccessMiddleware(projectService ProjectService, fileService FileService) *AccessMiddleware {
	return &AccessMiddleware{
		projectService: projectService,
		fileService:    fileService,
	}
}

func (m *AccessMiddleware) ProjectAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}

		projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid project ID"))
			return
		}

		p, err := m.projectService.Get(r.Context(), projectID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Project not found"))
			return
		}

		if !p.HasUser(userID) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Forbidden"))
			return
		}

		ctx := context.WithValue(r.Context(), projectContextKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AccessMiddleware) findFile(ctx context.Context, fileID uuid.UUID) (domainFile.File, error) {
	if tf, err := m.fileService.GetTypstFile(ctx, fileID); err == nil {
		return tf, nil
	}
	if bf, err := m.fileService.GetBinaryFile(ctx, fileID); err == nil {
		return bf, nil
	}
	return nil, domainFile.ErrFileNotFound
}

func (m *AccessMiddleware) FileAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}

		fileID, err := uuid.Parse(chi.URLParam(r, "fileID"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid file ID"))
			return
		}

		f, err := m.findFile(r.Context(), fileID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("File not found"))
			return
		}

		p, err := m.projectService.Get(r.Context(), f.ProjectID())
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Project not found"))
			return
		}

		if !p.HasUser(userID) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Forbidden"))
			return
		}

		ctx := context.WithValue(r.Context(), fileContextKey, f)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ProjectFromContext(ctx context.Context) (*domainProject.Project, bool) {
	p, ok := ctx.Value(projectContextKey).(*domainProject.Project)
	return p, ok
}

func WithProject(ctx context.Context, p *domainProject.Project) context.Context {
	return context.WithValue(ctx, projectContextKey, p)
}

func FileFromContext(ctx context.Context) (domainFile.File, bool) {
	f, ok := ctx.Value(fileContextKey).(domainFile.File)
	return f, ok
}

func WithFile(ctx context.Context, f domainFile.File) context.Context {
	return context.WithValue(ctx, fileContextKey, f)
}
