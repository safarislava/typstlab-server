package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	fileApp "github.com/safarislava/typstlab-server/internal/application/file"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
)

const docTyp = "doc.typ"

func testContext(userID uuid.UUID, project *domainProject.Project, file domainFile.File) context.Context {
	ctx := context.WithValue(context.Background(), userIDKey, userID)
	if project != nil {
		ctx = context.WithValue(ctx, projectContextKey, project)
	}
	if file != nil {
		ctx = context.WithValue(ctx, fileContextKey, file)
	}
	return ctx
}

func assertFileCreation(t *testing.T, rr *httptest.ResponseRecorder, expectedFileID uuid.UUID) {
	t.Helper()
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp JSONFileResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != expectedFileID.String() {
		t.Errorf("Expected file ID %q, got %q", expectedFileID.String(), resp.ID)
	}
}

type mockFileUseCase struct {
	fileApp.UseCase
	uploadTypstFileFunc    func(ctx context.Context, req *fileApp.UploadTypstFileRequest) (*domainFile.TypstFile, error)
	uploadBinaryFileFunc   func(ctx context.Context, req *fileApp.UploadBinaryFileRequest) (*domainFile.BinaryFile, error)
	getTypstFileFunc       func(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error)
	getBinaryFileFunc      func(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error)
	listFilesByProjectFunc func(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	applyFileChangesFunc   func(ctx context.Context, req fileApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
	deleteFileFunc         func(ctx context.Context, fileID uuid.UUID) error
}

func (m *mockFileUseCase) UploadTypstFile(ctx context.Context, req *fileApp.UploadTypstFileRequest) (*domainFile.TypstFile, error) {
	if m.uploadTypstFileFunc != nil {
		return m.uploadTypstFileFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockFileUseCase) UploadBinaryFile(ctx context.Context, req *fileApp.UploadBinaryFileRequest) (*domainFile.BinaryFile, error) {
	if m.uploadBinaryFileFunc != nil {
		return m.uploadBinaryFileFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockFileUseCase) GetTypstFile(ctx context.Context, fileID uuid.UUID) (*domainFile.TypstFile, error) {
	if m.getTypstFileFunc != nil {
		return m.getTypstFileFunc(ctx, fileID)
	}
	return nil, nil
}

func (m *mockFileUseCase) GetBinaryFile(ctx context.Context, fileID uuid.UUID) (*domainFile.BinaryFile, error) {
	if m.getBinaryFileFunc != nil {
		return m.getBinaryFileFunc(ctx, fileID)
	}
	return nil, nil
}

func (m *mockFileUseCase) ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	if m.listFilesByProjectFunc != nil {
		return m.listFilesByProjectFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockFileUseCase) ApplyFileChanges(ctx context.Context, req fileApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	if m.applyFileChangesFunc != nil {
		return m.applyFileChangesFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockFileUseCase) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, fileID)
	}
	return nil
}

func TestFileHandler_UploadTypstFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Test Project", time.Now())

	fileID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, nil, time.Now())

	mockFile := &mockFileUseCase{
		uploadTypstFileFunc: func(ctx context.Context, req *fileApp.UploadTypstFileRequest) (*domainFile.TypstFile, error) {
			if req.ID == fileID && req.ProjectID == projectID && req.Name == docTyp {
				return tf, nil
			}
			return nil, nil
		},
	}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, p, nil)

	reqBody, _ := json.Marshal(jsonUploadFileRequest{
		ID:   fileID.String(),
		Name: docTyp,
	})
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.UploadFile(rr, req)
	assertFileCreation(t, rr, fileID)
}

func TestFileHandler_UploadTypstFile_WithXML(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Test Project", time.Now())

	fileID := uuid.New()
	blockID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("state-bytes"), nil, time.Now())

	// valid XML format matching internal/infrastructure/serialization/xml_block.go
	xmlData := fmt.Sprintf(`<file state="c3RhdGUtYnl0ZXM="><block id=%q name="Intro">Content</block></file>`, blockID.String())

	mockFile := &mockFileUseCase{
		uploadTypstFileFunc: func(ctx context.Context, req *fileApp.UploadTypstFileRequest) (*domainFile.TypstFile, error) {
			if req.ID == fileID && req.ProjectID == projectID && req.Name == docTyp {
				if string(req.State) == "state-bytes" && len(req.Blocks) == 1 && req.Blocks[0].ID() == blockID {
					return tf, nil
				}
			}
			return nil, nil
		},
	}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, p, nil)

	reqBody, _ := json.Marshal(jsonUploadFileRequest{
		ID:      fileID.String(),
		Name:    docTyp,
		Content: []byte(xmlData),
	})
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.UploadFile(rr, req)
	assertFileCreation(t, rr, fileID)
}

func TestFileHandler_UploadBinaryFile_Multipart(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Test Project", time.Now())

	fileID := uuid.New()
	bf, _ := domainFile.NewBinaryFile(fileID, projectID, "img.png", []byte{1, 2, 3}, time.Now())

	mockFile := &mockFileUseCase{
		uploadBinaryFileFunc: func(ctx context.Context, req *fileApp.UploadBinaryFileRequest) (*domainFile.BinaryFile, error) {
			if req.ID == fileID && req.ProjectID == projectID && req.Name == "img.png" && bytes.Equal(req.Content, []byte{1, 2, 3}) {
				return bf, nil
			}
			return nil, nil
		},
	}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, p, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "img.png")
	_, _ = part.Write([]byte{1, 2, 3})
	_ = writer.WriteField("id", fileID.String())
	_ = writer.WriteField("name", "img.png")
	_ = writer.Close()

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/projects/"+projectID.String()+"/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	handler.UploadFile(rr, req)
	assertFileCreation(t, rr, fileID)
}

func TestFileHandler_ListProjectFiles(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Test Project", time.Now())

	tf, _ := domainFile.NewTypstFile(uuid.New(), projectID, docTyp, nil, nil, time.Now())
	mockFile := &mockFileUseCase{
		listFilesByProjectFunc: func(ctx context.Context, pid uuid.UUID) ([]domainFile.File, error) {
			return []domainFile.File{tf}, nil
		},
	}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, p, nil)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/projects/"+projectID.String()+"/files", nil)
	rr := httptest.NewRecorder()

	handler.ListProjectFiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp []JSONFileResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if len(resp) != 1 || resp[0].Name != docTyp {
		t.Errorf("Expected 1 file named doc.typ, got %+v", resp)
	}
}

func TestFileHandler_GetTypstFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("state"), nil, time.Now())
	mockFile := &mockFileUseCase{}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, nil, tf)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/typst/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	handler.GetTypstFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var resp JSONTypstFileResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Name != docTyp || string(resp.State) != "state" {
		t.Errorf("Unexpected response: %+v", resp)
	}
}

func TestFileHandler_GetBinaryFileRaw(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	bf, _ := domainFile.NewBinaryFile(fileID, projectID, "image.png", []byte{4, 5, 6}, time.Now())
	mockFile := &mockFileUseCase{}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, nil, bf)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/binary/"+fileID.String()+"/raw", nil)
	rr := httptest.NewRecorder()

	handler.GetBinaryFileRaw(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	if !bytes.Equal(rr.Body.Bytes(), []byte{4, 5, 6}) {
		t.Errorf("Expected body [4 5 6], got %v", rr.Body.Bytes())
	}
}

func TestFileHandler_ApplyFileChanges(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("old-state"), nil, time.Now())
	updatedTf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("updated-state"), nil, time.Now())

	mockFile := &mockFileUseCase{
		applyFileChangesFunc: func(ctx context.Context, req fileApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
			if req.FileID == fileID && string(req.Delta) == "changes" {
				return updatedTf, nil
			}
			return nil, nil
		},
	}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, nil, tf)

	reqBody, _ := json.Marshal(jsonApplyFileChangesRequest{Delta: []byte("changes")})
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/files/typst/"+fileID.String()+"/changes", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ApplyFileChanges(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp JSONTypstFileResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if string(resp.State) != "updated-state" {
		t.Errorf("Expected updated-state, got %q", resp.State)
	}
}

func TestFileHandler_DeleteFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Test Project", time.Now())

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, nil, time.Now())

	deletedFileCalled := false
	mockFile := &mockFileUseCase{
		deleteFileFunc: func(ctx context.Context, fid uuid.UUID) error {
			if fid == fileID {
				deletedFileCalled = true
			}
			return nil
		},
	}

	handler := NewFileHandler(mockFile)
	ctx := testContext(userID, p, tf)

	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/projects/"+projectID.String()+"/files/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	handler.DeleteFile(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusNoContent, rr.Code, rr.Body.String())
	}

	if !deletedFileCalled {
		t.Error("Expected file service delete to be called")
	}
}
