package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

type mockSyncService struct {
	syncFunc func(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error)
}

func (m *mockSyncService) Sync(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error) {
	if m.syncFunc != nil {
		return m.syncFunc(ctx, projectID, req)
	}
	return nil, errors.New("sync failed")
}

func TestSyncHandler_Sync_Success(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, err := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	clientFileID := uuid.New()
	syncSvc := &mockSyncService{
		syncFunc: func(ctx context.Context, pid uuid.UUID, req *syncApp.Request) (*syncApp.Response, error) {
			return &syncApp.Response{
				Instructions: []syncApp.Instruction{
					{
						Action: syncApp.ActionUpload,
						FileID: clientFileID,
					},
					{
						Action: syncApp.ActionDownload,
						FileID: uuid.New(),
					},
				},
			}, nil
		},
	}
	handler := NewHandler(syncSvc)

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

	ctx := middleware.WithProject(context.Background(), p)
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

	syncSvc := &mockSyncService{}
	handler := NewHandler(syncSvc)

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
	syncSvc := &mockSyncService{}
	handler := NewHandler(syncSvc)

	ctx := middleware.WithProject(context.Background(), p)
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
	syncSvc := &mockSyncService{}
	handler := NewHandler(syncSvc)

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

	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/sync", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
