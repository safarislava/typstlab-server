package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
)

const (
	errUnauthorized    = "unauthorized"
	errProjectNotFound = "project not found"
	errForbidden       = "forbidden"
	testProjectName    = "Test Project"
	docTypstName       = "doc.typ"
)

type mockProjectUseCaseForAccess struct {
	projectApp.UseCase
	getFunc func(ctx context.Context, projectID uuid.UUID) (*domainProject.Project, error)
}

func (m *mockProjectUseCaseForAccess) Get(ctx context.Context, projectID uuid.UUID) (*domainProject.Project, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, projectID)
	}
	return nil, errors.New(errProjectNotFound)
}

type mockFileUseCaseForAccess struct {
	fileApp.UseCase
	getTypstFileFunc  func(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	getBinaryFileFunc func(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error)
}

func (m *mockFileUseCaseForAccess) GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error) {
	if m.getTypstFileFunc != nil {
		return m.getTypstFileFunc(ctx, fileID)
	}
	return nil, errors.New("typst file not found")
}

func (m *mockFileUseCaseForAccess) GetBinaryFile(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error) {
	if m.getBinaryFileFunc != nil {
		return m.getBinaryFileFunc(ctx, fileID)
	}
	return nil, errors.New("binary file not found")
}

func TestNewAccessMiddleware(t *testing.T) {
	t.Parallel()

	projSvc := &mockProjectUseCaseForAccess{}
	fileSvc := &mockFileUseCaseForAccess{}

	mw := NewAccessMiddleware(projSvc, fileSvc)
	if mw == nil {
		t.Fatal("Expected non-nil AccessMiddleware")
	}
	if mw.projectService != projSvc {
		t.Error("Expected projectService to match injected mock")
	}
	if mw.fileService != fileSvc {
		t.Error("Expected fileService to match injected mock")
	}
}

func TestAccessMiddleware_ProjectAccess_Unauthorized(t *testing.T) {
	t.Parallel()

	mw := NewAccessMiddleware(&mockProjectUseCaseForAccess{}, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.ProjectAccess).Get("/projects/{projectID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/projects/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
	if rr.Body.String() != errUnauthorized {
		t.Errorf("Expected body %q, got %q", errUnauthorized, rr.Body.String())
	}
}

func TestAccessMiddleware_ProjectAccess_InvalidProjectID(t *testing.T) {
	t.Parallel()

	mw := NewAccessMiddleware(&mockProjectUseCaseForAccess{}, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.ProjectAccess).Get("/projects/{projectID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/not-a-uuid", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
	if rr.Body.String() != "Invalid project ID" {
		t.Errorf("Expected body 'Invalid project ID', got %q", rr.Body.String())
	}
}

func TestAccessMiddleware_ProjectAccess_ProjectNotFound(t *testing.T) {
	t.Parallel()

	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, projectID uuid.UUID) (*domainProject.Project, error) {
			return nil, errors.New("not found")
		},
	}

	mw := NewAccessMiddleware(projSvc, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.ProjectAccess).Get("/projects/{projectID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
	if rr.Body.String() != errProjectNotFound {
		t.Errorf("Expected body %q, got %q", errProjectNotFound, rr.Body.String())
	}
}

func TestAccessMiddleware_ProjectAccess_Forbidden(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{otherUserID}, testProjectName, time.Now())

	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
			if id == projectID {
				return p, nil
			}
			return nil, errors.New("not found")
		},
	}

	mw := NewAccessMiddleware(projSvc, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.ProjectAccess).Get("/projects/{projectID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
	if rr.Body.String() != errForbidden {
		t.Errorf("Expected body %q, got %q", errForbidden, rr.Body.String())
	}
}

func TestAccessMiddleware_ProjectAccess_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, testProjectName, time.Now())

	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, id uuid.UUID) (*domainProject.Project, error) {
			if id == projectID {
				return p, nil
			}
			return nil, errors.New("not found")
		},
	}

	var ctxProject *domainProject.Project
	var ok bool

	mw := NewAccessMiddleware(projSvc, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.ProjectAccess).Get("/projects/{projectID}", func(w http.ResponseWriter, r *http.Request) {
		ctxProject, ok = ProjectFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	if !ok {
		t.Fatal("Expected project in context")
	}
	if ctxProject.ID() != projectID {
		t.Errorf("Expected project ID %s, got %s", projectID, ctxProject.ID())
	}
}

func TestAccessMiddleware_FileAccess_Unauthorized(t *testing.T) {
	t.Parallel()

	mw := NewAccessMiddleware(&mockProjectUseCaseForAccess{}, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/files/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
	if rr.Body.String() != errUnauthorized {
		t.Errorf("Expected body %q, got %q", errUnauthorized, rr.Body.String())
	}
}

func TestAccessMiddleware_FileAccess_InvalidFileID(t *testing.T) {
	t.Parallel()

	mw := NewAccessMiddleware(&mockProjectUseCaseForAccess{}, &mockFileUseCaseForAccess{})
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
	if rr.Body.String() != "Invalid file ID" {
		t.Errorf("Expected body 'Invalid file ID', got %q", rr.Body.String())
	}
}

func TestAccessMiddleware_FileAccess_FileNotFound(t *testing.T) {
	t.Parallel()

	fileSvc := &mockFileUseCaseForAccess{
		getTypstFileFunc: func(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error) {
			return nil, errors.New("not found")
		},
		getBinaryFileFunc: func(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error) {
			return nil, errors.New("not found")
		},
	}

	mw := NewAccessMiddleware(&mockProjectUseCaseForAccess{}, fileSvc)
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
	if rr.Body.String() != "File not found" {
		t.Errorf("Expected body 'File not found', got %q", rr.Body.String())
	}
}

func TestAccessMiddleware_FileAccess_ProjectNotFound(t *testing.T) {
	t.Parallel()

	fileID := uuid.New()
	projectID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTypstName, nil, nil, time.Now())

	fileSvc := &mockFileUseCaseForAccess{
		getTypstFileFunc: func(ctx context.Context, fid uuid.UUID) (*domainFile.TypstFile, error) {
			if fid == fileID {
				return tf, nil
			}
			return nil, errors.New("not found")
		},
	}
	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, pid uuid.UUID) (*domainProject.Project, error) {
			return nil, errors.New(errProjectNotFound)
		},
	}

	mw := NewAccessMiddleware(projSvc, fileSvc)
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
	if rr.Body.String() != errProjectNotFound {
		t.Errorf("Expected body %q, got %q", errProjectNotFound, rr.Body.String())
	}
}

func TestAccessMiddleware_FileAccess_Forbidden(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTypstName, nil, nil, time.Now())
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{otherUserID}, testProjectName, time.Now())

	fileSvc := &mockFileUseCaseForAccess{
		getTypstFileFunc: func(ctx context.Context, fid uuid.UUID) (*domainFile.TypstFile, error) {
			if fid == fileID {
				return tf, nil
			}
			return nil, errors.New("not found")
		},
	}
	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, pid uuid.UUID) (*domainProject.Project, error) {
			if pid == projectID {
				return p, nil
			}
			return nil, errors.New("not found")
		},
	}

	mw := NewAccessMiddleware(projSvc, fileSvc)
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
	if rr.Body.String() != errForbidden {
		t.Errorf("Expected body %q, got %q", errForbidden, rr.Body.String())
	}
}

func TestAccessMiddleware_FileAccess_Success_TypstFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTypstName, nil, nil, time.Now())
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, testProjectName, time.Now())

	fileSvc := &mockFileUseCaseForAccess{
		getTypstFileFunc: func(ctx context.Context, fid uuid.UUID) (*domainFile.TypstFile, error) {
			if fid == fileID {
				return tf, nil
			}
			return nil, errors.New("not found")
		},
	}
	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, pid uuid.UUID) (*domainProject.Project, error) {
			if pid == projectID {
				return p, nil
			}
			return nil, errors.New("not found")
		},
	}

	var ctxFile domainFile.File
	var ok bool

	mw := NewAccessMiddleware(projSvc, fileSvc)
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		ctxFile, ok = FileFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	if !ok {
		t.Fatal("Expected file in context")
	}
	if ctxFile.ID() != fileID {
		t.Errorf("Expected file ID %s, got %s", fileID, ctxFile.ID())
	}
}

func TestAccessMiddleware_FileAccess_Success_BinaryFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	bf, _ := domainFile.NewBinaryFile(fileID, projectID, "image.png", []byte{1, 2, 3}, time.Now())
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, testProjectName, time.Now())

	fileSvc := &mockFileUseCaseForAccess{
		getTypstFileFunc: func(ctx context.Context, fid uuid.UUID) (*domainFile.TypstFile, error) {
			return nil, errors.New("not a typst file")
		},
		getBinaryFileFunc: func(ctx context.Context, fid uuid.UUID) (*domainFile.BinaryFile, error) {
			if fid == fileID {
				return bf, nil
			}
			return nil, errors.New("not found")
		},
	}
	projSvc := &mockProjectUseCaseForAccess{
		getFunc: func(ctx context.Context, pid uuid.UUID) (*domainProject.Project, error) {
			if pid == projectID {
				return p, nil
			}
			return nil, errors.New("not found")
		},
	}

	var ctxFile domainFile.File
	var ok bool

	mw := NewAccessMiddleware(projSvc, fileSvc)
	r := chi.NewRouter()
	r.With(mw.FileAccess).Get("/files/{fileID}", func(w http.ResponseWriter, r *http.Request) {
		ctxFile, ok = FileFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), userIDKey, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	if !ok {
		t.Fatal("Expected file in context")
	}
	if ctxFile.ID() != fileID {
		t.Errorf("Expected file ID %s, got %s", fileID, ctxFile.ID())
	}
}

func TestProjectFromContext_EmptyContext(t *testing.T) {
	t.Parallel()

	p, ok := ProjectFromContext(context.Background())
	if ok {
		t.Error("Expected ok to be false for empty context")
	}
	if p != nil {
		t.Error("Expected nil project for empty context")
	}
}

func TestProjectFromContext_InvalidContextValueType(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), projectContextKey, "invalid-type")
	p, ok := ProjectFromContext(ctx)
	if ok {
		t.Error("Expected ok to be false for wrong context type")
	}
	if p != nil {
		t.Error("Expected nil project for wrong context type")
	}
}

func TestProjectFromContext_ValidProjectContext(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	expectedProject, _ := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, testProjectName, time.Now())
	ctx := context.WithValue(context.Background(), projectContextKey, expectedProject)

	p, ok := ProjectFromContext(ctx)
	if !ok {
		t.Fatal("Expected ok to be true for valid project context")
	}
	if p != expectedProject {
		t.Errorf("Expected project %v, got %v", expectedProject, p)
	}
}

func TestFileFromContext_EmptyContext(t *testing.T) {
	t.Parallel()

	f, ok := FileFromContext(context.Background())
	if ok {
		t.Error("Expected ok to be false for empty context")
	}
	if f != nil {
		t.Error("Expected nil file for empty context")
	}
}

func TestFileFromContext_InvalidContextValueType(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), fileContextKey, 12345)
	f, ok := FileFromContext(ctx)
	if ok {
		t.Error("Expected ok to be false for wrong context type")
	}
	if f != nil {
		t.Error("Expected nil file for wrong context type")
	}
}

func TestFileFromContext_ValidFileContext(t *testing.T) {
	t.Parallel()

	fileID := uuid.New()
	expectedFile, _ := domainFile.NewTypstFile(fileID, uuid.New(), docTypstName, nil, nil, time.Now())
	ctx := context.WithValue(context.Background(), fileContextKey, expectedFile)

	f, ok := FileFromContext(ctx)
	if !ok {
		t.Fatal("Expected ok to be true for valid file context")
	}
	if f != expectedFile {
		t.Errorf("Expected file %v, got %v", expectedFile, f)
	}
}
