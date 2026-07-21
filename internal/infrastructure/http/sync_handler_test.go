package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
)

type mockSyncRepo struct {
	files []domainFile.File
}

func (m *mockSyncRepo) SaveTypstFile(context.Context, *domainFile.TypstFile) error {
	return nil
}

func (m *mockSyncRepo) SaveBinaryFile(context.Context, *domainFile.BinaryFile) error {
	return nil
}

func (m *mockSyncRepo) FindTypstFileByID(context.Context, uuid.UUID) (*domainFile.TypstFile, error) {
	return nil, nil
}

func (m *mockSyncRepo) FindBinaryFileByID(context.Context, uuid.UUID) (*domainFile.BinaryFile, error) {
	return nil, nil
}

func (m *mockSyncRepo) FindByProjectID(context.Context, uuid.UUID) ([]domainFile.File, error) {
	return m.files, nil
}

func (m *mockSyncRepo) DeleteFile(context.Context, uuid.UUID) error {
	return nil
}

func (m *mockSyncRepo) IsDeleted(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func TestSyncHandler_Sync_Success(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, err := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	serverFileID := uuid.New()
	serverFile, _ := domainFile.NewTypstFile(serverFileID, projectID, "server.typ", nil, nil, time.Now())
	repo := &mockSyncRepo{files: []domainFile.File{serverFile}}
	syncSvc := syncApp.NewService(repo)
	handler := NewSyncHandler(syncSvc)

	clientFileID := uuid.New()
	jsonReq := JSONSyncRequest{
		Files: []JSONSyncFileRequest{
			{
				ID:   clientFileID.String(),
				Name: "offline.typ",
				Type: string(domainFile.TypeTypst),
			},
		},
	}
	body, _ := json.Marshal(jsonReq)

	ctx := context.WithValue(context.Background(), projectContextKey, p)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/sync", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var jsonResp JSONSyncResponse
	if err := json.NewDecoder(rr.Body).Decode(&jsonResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(jsonResp.Instructions) < 2 {
		t.Errorf("Expected at least 2 instructions (upload for offline file, download for server file), got %d", len(jsonResp.Instructions))
	}
}

func TestSyncHandler_Sync_MissingProjectContext(t *testing.T) {
	t.Parallel()

	syncSvc := syncApp.NewService(&mockSyncRepo{})
	handler := NewSyncHandler(syncSvc)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/sync", nil)
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestSyncHandler_Sync_InvalidJSON(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	syncSvc := syncApp.NewService(&mockSyncRepo{})
	handler := NewSyncHandler(syncSvc)

	ctx := context.WithValue(context.Background(), projectContextKey, p)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/sync", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestSyncHandler_Sync_InvalidFileID(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	syncSvc := syncApp.NewService(&mockSyncRepo{})
	handler := NewSyncHandler(syncSvc)

	jsonReq := JSONSyncRequest{
		Files: []JSONSyncFileRequest{
			{
				ID:   "invalid-uuid",
				Name: "test.typ",
				Type: string(domainFile.TypeTypst),
			},
		},
	}
	body, _ := json.Marshal(jsonReq)

	ctx := context.Background()
	ctx = context.WithValue(ctx, projectContextKey, p)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/sync", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestSyncHandler_Sync_FullIntegration(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, err := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Integration Project", time.Now())
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	serverFileID := uuid.New()
	deletedFileID := uuid.New()
	serverFile, _ := domainFile.NewTypstFile(serverFileID, projectID, "main.typ", nil, nil, time.Now())

	repo := &mockSyncRepo{files: []domainFile.File{serverFile}}
	syncSvc := syncApp.NewService(repo)
	handler := NewSyncHandler(syncSvc)

	offlineID := uuid.New()
	jsonReq := JSONSyncRequest{
		Files: []JSONSyncFileRequest{
			{
				ID:   offlineID.String(),
				Name: "offline_created.typ",
				Type: string(domainFile.TypeTypst),
			},
			{
				ID:   deletedFileID.String(),
				Name: "deleted.typ",
				Type: string(domainFile.TypeTypst),
			},
		},
	}
	body, _ := json.Marshal(jsonReq)

	ctx := context.WithValue(context.Background(), projectContextKey, p)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/sync", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp JSONSyncResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	actions := make(map[string]bool)
	for _, inst := range resp.Instructions {
		actions[inst.Action] = true
	}

	if !actions["upload"] || !actions["download"] {
		t.Errorf("Expected upload and download actions in response, got instructions: %+v", resp.Instructions)
	}
}
