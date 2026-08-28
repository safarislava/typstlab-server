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
		ContentVectors: map[string][]byte{
			clientFileID.String(): []byte("vector-state"),
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

	if len(jsonResp.Instructions) != 2 {
		t.Errorf("Expected 2 instructions, got %d", len(jsonResp.Instructions))
	}
}

func TestSyncHandler_Sync_WithMetadata(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	p, err := domainProject.NewProject(projectID, []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	mockDelta := []byte("server-metadata-delta")
	syncSvc := &mockSyncService{
		syncFunc: func(ctx context.Context, pid uuid.UUID, req *syncApp.Request) (*syncApp.Response, error) {
			if string(req.MetadataStateVector) != "client-state-vector" {
				t.Errorf("unexpected metadata state vector: %s", string(req.MetadataStateVector))
			}
			return &syncApp.Response{
				MetadataDelta: mockDelta,
			}, nil
		},
	}
	handler := NewHandler(syncSvc)

	jsonReq := JSONSyncRequest{
		MetadataStateVector: []byte("client-state-vector"),
	}
	body, _ := json.Marshal(jsonReq)

	ctx := middleware.WithProject(context.Background(), p)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/sync", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var jsonResp JSONSyncResponse
	if err := json.NewDecoder(rr.Body).Decode(&jsonResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !bytes.Equal(jsonResp.MetadataDelta, mockDelta) {
		t.Errorf("expected metadata delta %s, got %s", string(mockDelta), string(jsonResp.MetadataDelta))
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
		ContentVectors: map[string][]byte{
			"invalid-uuid": []byte("some-vector"),
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
