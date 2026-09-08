package file

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	binaryApp "github.com/safarislava/typstlab-server/internal/application/file/binary"
	typstApp "github.com/safarislava/typstlab-server/internal/application/file/typst"
	syncApp "github.com/safarislava/typstlab-server/internal/application/sync"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
	domainProject "github.com/safarislava/typstlab-server/internal/domain/project"
	"github.com/safarislava/typstlab-server/internal/infrastructure/http/middleware"
)

const (
	docTyp     = "doc.typxml"
	testTypxml = "test.typxml"
)

func testContext(userID uuid.UUID, project *domainProject.Project, file domainFile.File) context.Context {
	ctx := middleware.WithUserID(context.Background(), userID)
	if project != nil {
		ctx = middleware.WithProject(ctx, project)
	}
	if file != nil {
		ctx = middleware.WithFile(ctx, file)
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

type mockTypstService struct {
	uploadFunc func(ctx context.Context, req *typstApp.UploadRequest) (*domainFile.TypstFile, error)
}

func (m *mockTypstService) Upload(ctx context.Context, req *typstApp.UploadRequest) (*domainFile.TypstFile, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, req)
	}
	return nil, nil
}

type mockBinaryService struct {
	uploadFunc func(ctx context.Context, req *binaryApp.UploadRequest) (*domainFile.BinaryFile, error)
}

func (m *mockBinaryService) Upload(ctx context.Context, req *binaryApp.UploadRequest) (*domainFile.BinaryFile, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, req)
	}
	return nil, nil
}

type mockFileService struct {
	listFilesByProjectFunc func(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error)
	deleteFileFunc         func(ctx context.Context, fileID uuid.UUID) error
}

func (m *mockFileService) ListFilesByProject(ctx context.Context, projectID uuid.UUID) ([]domainFile.File, error) {
	if m.listFilesByProjectFunc != nil {
		return m.listFilesByProjectFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockFileService) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, fileID)
	}
	return nil
}

type mockChangeApplier struct {
	applyFileChangesFunc func(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error)
}

func (m *mockChangeApplier) ApplyFileChanges(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
	if m.applyFileChangesFunc != nil {
		return m.applyFileChangesFunc(ctx, req)
	}
	return nil, nil
}

func TestFileHandler_UploadTypstFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Test Project", time.Now())

	fileID := uuid.New()
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, time.Now())

	mockTypst := &mockTypstService{
		uploadFunc: func(ctx context.Context, req *typstApp.UploadRequest) (*domainFile.TypstFile, error) {
			if req.ID == fileID && req.ProjectID == projectID && req.Name == docTyp {
				return tf, nil
			}
			return nil, nil
		},
	}

	handler := NewHandler(mockTypst, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})
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
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("state-bytes"), time.Now())

	xmlData := `<file state="c3RhdGUtYnl0ZXM="></file>`

	mockTypst := &mockTypstService{
		uploadFunc: func(ctx context.Context, req *typstApp.UploadRequest) (*domainFile.TypstFile, error) {
			if req.ID == fileID && req.ProjectID == projectID && req.Name == docTyp {
				if string(req.State) == "state-bytes" {
					return tf, nil
				}
			}
			return nil, nil
		},
	}

	handler := NewHandler(mockTypst, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})
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

	mockBinary := &mockBinaryService{
		uploadFunc: func(ctx context.Context, req *binaryApp.UploadRequest) (*domainFile.BinaryFile, error) {
			if req.ID == fileID && req.ProjectID == projectID && req.Name == "img.png" && bytes.Equal(req.Content, []byte{1, 2, 3}) {
				return bf, nil
			}
			return nil, nil
		},
	}

	handler := NewHandler(&mockTypstService{}, mockBinary, &mockFileService{}, &mockChangeApplier{})
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

	tf, _ := domainFile.NewTypstFile(uuid.New(), projectID, docTyp, nil, time.Now())
	mockFile := &mockFileService{
		listFilesByProjectFunc: func(ctx context.Context, pid uuid.UUID) ([]domainFile.File, error) {
			return []domainFile.File{tf}, nil
		},
	}

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, mockFile, &mockChangeApplier{})
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

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("state"), time.Now())

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})
	ctx := testContext(userID, nil, tf)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/typst/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	handler.GetTypstFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
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

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})
	ctx := testContext(userID, nil, bf)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/binary/"+fileID.String()+"/raw", nil)
	rr := httptest.NewRecorder()

	handler.GetBinaryFileRaw(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
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

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("old-state"), time.Now())
	updatedTf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, []byte("updated-state"), time.Now())

	mockChange := &mockChangeApplier{
		applyFileChangesFunc: func(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
			if req.FileID == fileID && string(req.Delta) == "changes" {
				return updatedTf, nil
			}
			return nil, nil
		},
	}

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, mockChange)
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

	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, time.Now())

	deletedFileCalled := false
	mockFile := &mockFileService{
		deleteFileFunc: func(ctx context.Context, fid uuid.UUID) error {
			if fid == fileID {
				deletedFileCalled = true
			}
			return nil
		},
	}

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, mockFile, &mockChangeApplier{})
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

func TestFileHandler_GetBinaryFile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	bf, _ := domainFile.NewBinaryFile(fileID, projectID, "image.png", []byte{1, 2, 3}, time.Now())

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})
	ctx := testContext(userID, nil, bf)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/binary/"+fileID.String(), nil)
	rr := httptest.NewRecorder()

	handler.GetBinaryFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp JSONBinaryFileResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if resp.ID != fileID.String() || resp.Name != "image.png" || resp.Size != 3 {
		t.Errorf("Unexpected response: %+v", resp)
	}
}

func TestFileHandler_GetTypstFile_Errors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})

	// Case 1: Missing file context
	req1 := httptest.NewRequestWithContext(testContext(userID, nil, nil), http.MethodGet, "/files/typst/"+fileID.String(), nil)
	rr1 := httptest.NewRecorder()
	handler.GetTypstFile(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: File is binary, not typst
	bf, _ := domainFile.NewBinaryFile(fileID, projectID, "img.png", []byte{1}, time.Now())
	req2 := httptest.NewRequestWithContext(testContext(userID, nil, bf), http.MethodGet, "/files/typst/"+fileID.String(), nil)
	rr2 := httptest.NewRecorder()
	handler.GetTypstFile(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr2.Code)
	}
}

func TestFileHandler_GetBinaryFile_Errors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})

	// Case 1: Missing file context
	req1 := httptest.NewRequestWithContext(testContext(userID, nil, nil), http.MethodGet, "/files/binary/"+fileID.String(), nil)
	rr1 := httptest.NewRecorder()
	handler.GetBinaryFile(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: File is typst, not binary
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, time.Now())
	req2 := httptest.NewRequestWithContext(testContext(userID, nil, tf), http.MethodGet, "/files/binary/"+fileID.String(), nil)
	rr2 := httptest.NewRecorder()
	handler.GetBinaryFile(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr2.Code)
	}
}

func TestFileHandler_GetBinaryFileRaw_Errors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})

	// Case 1: Missing file context
	req1 := httptest.NewRequestWithContext(testContext(userID, nil, nil), http.MethodGet, "/files/binary/"+fileID.String()+"/raw", nil)
	rr1 := httptest.NewRecorder()
	handler.GetBinaryFileRaw(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: File is typst, not binary
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, time.Now())
	req2 := httptest.NewRequestWithContext(testContext(userID, nil, tf), http.MethodGet, "/files/binary/"+fileID.String()+"/raw", nil)
	rr2 := httptest.NewRecorder()
	handler.GetBinaryFileRaw(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr2.Code)
	}
}

func TestFileHandler_ApplyFileChanges_Errors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	fileID := uuid.New()

	mockChange := &mockChangeApplier{
		applyFileChangesFunc: func(ctx context.Context, req syncApp.ApplyFileChangesRequest) (*domainFile.TypstFile, error) {
			return nil, errors.New("apply error")
		},
	}
	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, mockChange)

	// Case 1: Missing file context
	req1 := httptest.NewRequestWithContext(testContext(userID, nil, nil), http.MethodPost, "/files/typst/"+fileID.String()+"/changes", bytes.NewBufferString("{}"))
	rr1 := httptest.NewRecorder()
	handler.ApplyFileChanges(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: Not a Typst file
	bf, _ := domainFile.NewBinaryFile(fileID, projectID, "img.png", []byte{1}, time.Now())
	req2 := httptest.NewRequestWithContext(testContext(userID, nil, bf), http.MethodPost, "/files/typst/"+fileID.String()+"/changes", bytes.NewBufferString("{}"))
	rr2 := httptest.NewRecorder()
	handler.ApplyFileChanges(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr2.Code)
	}

	// Case 3: Invalid body JSON
	tf, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, time.Now())
	req3 := httptest.NewRequestWithContext(testContext(userID, nil, tf), http.MethodPost, "/files/typst/"+fileID.String()+"/changes", bytes.NewBufferString("invalid json"))
	rr3 := httptest.NewRecorder()
	handler.ApplyFileChanges(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr3.Code)
	}

	// Case 4: Service error
	reqBody, _ := json.Marshal(jsonApplyFileChangesRequest{Delta: []byte("delta")})
	req4 := httptest.NewRequestWithContext(testContext(userID, nil, tf), http.MethodPost, "/files/typst/"+fileID.String()+"/changes", bytes.NewBuffer(reqBody))
	rr4 := httptest.NewRecorder()
	handler.ApplyFileChanges(rr4, req4)
	if rr4.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr4.Code)
	}
}

func TestFileHandler_DeleteFile_Errors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	otherProjectID := uuid.New()
	fileID := uuid.New()

	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Project", time.Now())
	tf, _ := domainFile.NewTypstFile(fileID, otherProjectID, docTyp, nil, time.Now())

	mockFile := &mockFileService{
		deleteFileFunc: func(ctx context.Context, fid uuid.UUID) error {
			return errors.New("delete error")
		},
	}
	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, mockFile, &mockChangeApplier{})

	// Case 1: Missing context
	req1 := httptest.NewRequestWithContext(testContext(userID, nil, nil), http.MethodDelete, "/projects/"+projectID.String()+"/files/"+fileID.String(), nil)
	rr1 := httptest.NewRecorder()
	handler.DeleteFile(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: Project mismatch
	req2 := httptest.NewRequestWithContext(testContext(userID, p, tf), http.MethodDelete, "/projects/"+projectID.String()+"/files/"+fileID.String(), nil)
	rr2 := httptest.NewRecorder()
	handler.DeleteFile(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr2.Code)
	}

	// Case 3: Service error
	matchingTF, _ := domainFile.NewTypstFile(fileID, projectID, docTyp, nil, time.Now())
	req3 := httptest.NewRequestWithContext(testContext(userID, p, matchingTF), http.MethodDelete, "/projects/"+projectID.String()+"/files/"+fileID.String(), nil)
	rr3 := httptest.NewRecorder()
	handler.DeleteFile(rr3, req3)
	if rr3.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr3.Code)
	}
}

func TestFileHandler_UploadFile_RequestErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Project", time.Now())

	handler := NewHandler(&mockTypstService{}, &mockBinaryService{}, &mockFileService{}, &mockChangeApplier{})

	// Case 1: Missing project context
	req1 := httptest.NewRequestWithContext(testContext(userID, nil, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBufferString("{}"))
	rr1 := httptest.NewRecorder()
	handler.UploadFile(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: Unsupported Content-Type
	req2 := httptest.NewRequestWithContext(testContext(userID, p, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBufferString("text"))
	req2.Header.Set("Content-Type", "text/plain")
	rr2 := httptest.NewRecorder()
	handler.UploadFile(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr2.Code)
	}

	// Case 3: Invalid JSON body
	req3 := httptest.NewRequestWithContext(testContext(userID, p, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBufferString("invalid json"))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	handler.UploadFile(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr3.Code)
	}

	// Case 4: Empty file name
	reqBodyEmptyName, _ := json.Marshal(jsonUploadFileRequest{
		ID:   uuid.New().String(),
		Name: "",
	})
	req4Empty := httptest.NewRequestWithContext(testContext(userID, p, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBuffer(reqBodyEmptyName))
	req4Empty.Header.Set("Content-Type", "application/json")
	rr4Empty := httptest.NewRecorder()
	handler.UploadFile(rr4Empty, req4Empty)
	if rr4Empty.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr4Empty.Code)
	}
}

func TestFileHandler_UploadFile_ServiceErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	projectID := uuid.New()
	p, _ := domainProject.NewProject(projectID, []uuid.UUID{userID}, "Project", time.Now())

	mockTypst := &mockTypstService{
		uploadFunc: func(ctx context.Context, req *typstApp.UploadRequest) (*domainFile.TypstFile, error) {
			return nil, errors.New("upload typst error")
		},
	}
	mockBinary := &mockBinaryService{
		uploadFunc: func(ctx context.Context, req *binaryApp.UploadRequest) (*domainFile.BinaryFile, error) {
			return nil, errors.New("upload binary error")
		},
	}
	handler := NewHandler(mockTypst, mockBinary, &mockFileService{}, &mockChangeApplier{})

	// Case 1: Upload typst service error
	reqBody, _ := json.Marshal(jsonUploadFileRequest{
		ID:   uuid.New().String(),
		Name: testTypxml,
	})
	req1 := httptest.NewRequestWithContext(testContext(userID, p, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBuffer(reqBody))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	handler.UploadFile(rr1, req1)
	if rr1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr1.Code)
	}

	// Case 2: Upload binary service error
	reqBodyBin, _ := json.Marshal(jsonUploadFileRequest{
		ID:   uuid.New().String(),
		Name: "test.png",
	})
	req2 := httptest.NewRequestWithContext(testContext(userID, p, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBuffer(reqBodyBin))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handler.UploadFile(rr2, req2)
	if rr2.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr2.Code)
	}

	// Case 3: Invalid typxml XML payload
	reqBodyInvalidXML, _ := json.Marshal(jsonUploadFileRequest{
		ID:      uuid.New().String(),
		Name:    testTypxml,
		Content: []byte("<invalid-xml>"),
	})
	req3 := httptest.NewRequestWithContext(testContext(userID, p, nil), http.MethodPost, "/projects/"+projectID.String()+"/files", bytes.NewBuffer(reqBodyInvalidXML))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	handler.UploadFile(rr3, req3)
	if rr3.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr3.Code)
	}
}
