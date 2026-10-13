package collaboration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type mockSyncService struct {
	syncFunc             func(ctx context.Context, projectID uuid.UUID, req *syncApp.Request) (*syncApp.Response, error)
	applyFileChangesFunc func(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
}

func (m *mockSyncService) Sync(
	ctx context.Context,
	projectID uuid.UUID,
	req *syncApp.Request,
) (*syncApp.Response, error) {
	if m.syncFunc != nil {
		return m.syncFunc(ctx, projectID, req)
	}
	return nil, nil
}

func (m *mockSyncService) ApplyFileChanges(
	ctx context.Context,
	req syncApp.ApplyFileChangesRequest,
) (*domainFile.TypstFile, error) {
	if m.applyFileChangesFunc != nil {
		return m.applyFileChangesFunc(ctx, req)
	}
	return nil, nil
}

func TestService_SyncInit(t *testing.T) {
	t.Parallel()

	fileID := uuid.New()
	mockSync := &mockSyncService{
		syncFunc: func(_ context.Context, _ uuid.UUID, req *syncApp.Request) (*syncApp.Response, error) {
			if string(req.MetadataStateVector) != "state-vector" {
				t.Errorf("unexpected metadata state vector: %s", string(req.MetadataStateVector))
			}
			return &syncApp.Response{
				MetadataDelta: []byte("server-delta"),
				Instructions: []syncApp.Instruction{
					{
						Action: syncApp.ActionApplyChanges,
						FileID: fileID,
						Delta:  []byte("diff"),
					},
				},
			}, nil
		},
	}

	service := NewService(mockSync)
	projectID := uuid.New()

	req := SyncInitRequest{
		MetadataStateVector: []byte("state-vector"),
		ContentVectors:      map[uuid.UUID][]byte{fileID: []byte("vector")},
	}

	resp, err := service.SyncInit(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(resp.MetadataDelta) != "server-delta" {
		t.Errorf("expected server-delta, got %s", string(resp.MetadataDelta))
	}
	if len(resp.Instructions) != 1 || resp.Instructions[0].FileID != fileID {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestService_ApplyCRDTUpdate(t *testing.T) {
	t.Parallel()

	applied := false
	fileID := uuid.New()
	mockSync := &mockSyncService{
		applyFileChangesFunc: func(_ context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
			if req.FileID == fileID && string(req.Delta) == "crdt-delta" {
				applied = true
			}
			return nil, nil
		},
	}

	service := NewService(mockSync)

	if err := service.ApplyCRDTUpdate(context.Background(), fileID, []byte("crdt-delta")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !applied {
		t.Error("expected applyFileChanges to be called")
	}

	// Validation errors
	if err := service.ApplyCRDTUpdate(context.Background(), uuid.Nil, []byte("crdt-delta")); err == nil {
		t.Error("expected error for nil file_id")
	}
	if err := service.ApplyCRDTUpdate(context.Background(), fileID, nil); err == nil {
		t.Error("expected error for empty delta")
	}
}

func TestService_ApplyMetadataUpdate(t *testing.T) {
	t.Parallel()

	synced := false
	mockSync := &mockSyncService{
		syncFunc: func(_ context.Context, _ uuid.UUID, req *syncApp.Request) (*syncApp.Response, error) {
			if string(req.MetadataDelta) == "meta-update" {
				synced = true
			}
			return &syncApp.Response{}, nil
		},
	}

	service := NewService(mockSync)
	projectID := uuid.New()

	if err := service.ApplyMetadataUpdate(context.Background(), projectID, []byte("meta-update")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !synced {
		t.Error("expected sync to be called")
	}

	// Validation error
	if err := service.ApplyMetadataUpdate(context.Background(), projectID, nil); err == nil {
		t.Error("expected error for empty delta")
	}
}

func TestService_Errors(t *testing.T) {
	t.Parallel()

	mockSync := &mockSyncService{
		syncFunc: func(_ context.Context, _ uuid.UUID, _ *syncApp.Request) (*syncApp.Response, error) {
			return nil, errors.New("database err")
		},
		applyFileChangesFunc: func(_ context.Context, _ syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
			return nil, errors.New("crdt merge err")
		},
	}

	service := NewService(mockSync)
	projectID := uuid.New()
	fileID := uuid.New()

	if _, err := service.SyncInit(context.Background(), projectID, SyncInitRequest{}); err == nil {
		t.Error("expected sync init error")
	}

	if err := service.ApplyCRDTUpdate(context.Background(), fileID, []byte("data")); err == nil {
		t.Error("expected apply crdt error")
	}

	if err := service.ApplyMetadataUpdate(context.Background(), projectID, []byte("data")); err == nil {
		t.Error("expected apply metadata error")
	}
}
