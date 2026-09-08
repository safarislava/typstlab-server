package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	testBucket    = "typstlab-test"
	testAccessKey = "minioadmin"
	testSecretKey = "minioadmin"
	mockAuth      = "test"
	mockBucket    = "test-bucket"
)

func TestNewClient_Validation(t *testing.T) {
	t.Parallel()

	_, err := NewClient(nil)
	if !errors.Is(err, ErrConfigRequired) {
		t.Errorf("expected ErrConfigRequired, got %v", err)
	}

	_, err = NewClient(&Config{})
	if !errors.Is(err, ErrEndpointRequired) {
		t.Errorf("expected ErrEndpointRequired, got %v", err)
	}

	_, err = NewClient(&Config{Endpoint: "localhost:9000"})
	if !errors.Is(err, ErrBucketRequired) {
		t.Errorf("expected ErrBucketRequired, got %v", err)
	}
}

func TestNewClient_Success(t *testing.T) {
	t.Parallel()

	client, err := NewClient(&Config{
		Endpoint:  "http://localhost:9000",
		Bucket:    testBucket,
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil || client.bucket != testBucket {
		t.Errorf("unexpected client: %+v", client)
	}
}

func TestBlobKey(t *testing.T) {
	t.Parallel()

	pID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	fID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	key := BlobKey(pID, fID, "image.png")

	expected := "projects/10000000-0000-0000-0000-000000000001/files/20000000-0000-0000-0000-000000000002_image.png"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func newMockS3Server(recordedObject *[]byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
			return
		}
		handleMockS3Methods(w, r, recordedObject)
	}))
}

func handleMockS3Methods(w http.ResponseWriter, r *http.Request, recordedObject *[]byte) {
	switch r.Method {
	case http.MethodHead:
		if strings.Contains(r.URL.Path, "nonexistent") || strings.Contains(r.URL.Path, "new-bucket") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		setMockS3Headers(w, "5")
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		data, _ := io.ReadAll(r.Body)
		if recordedObject != nil {
			*recordedObject = data
		}
		w.Header().Set("ETag", `"d41d8cd98f00b204e9800998ecf8427e"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		setMockS3Headers(w, "5")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func setMockS3Headers(w http.ResponseWriter, length string) {
	w.Header().Set("Content-Length", length)
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", `"d41d8cd98f00b204e9800998ecf8427e"`)
	w.Header().Set("Content-Type", "image/png")
}

func TestClient_UploadDownload(t *testing.T) {
	t.Parallel()

	var recordedObject []byte
	server := newMockS3Server(&recordedObject)
	defer server.Close()

	client, err := NewClient(&Config{
		Endpoint:  server.URL,
		Bucket:    mockBucket,
		AccessKey: mockAuth,
		SecretKey: mockAuth,
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// 1. Upload
	content := []byte("hello s3")
	uploadErr := client.Upload(ctx, "test.png", bytes.NewReader(content), int64(len(content)), "image/png")
	if uploadErr != nil {
		t.Fatalf("upload failed: %v", uploadErr)
	}
	if !strings.Contains(string(recordedObject), "hello s3") {
		t.Errorf("expected uploaded body to contain 'hello s3', got %s", recordedObject)
	}

	// 2. Download
	rc, dlErr := client.Download(ctx, "test.png")
	if dlErr != nil {
		t.Fatalf("download failed: %v", dlErr)
	}
	defer func() { _ = rc.Close() }()
	downloaded, _ := io.ReadAll(rc)
	if string(downloaded) != "hello" {
		t.Errorf("expected downloaded 'hello', got %s", downloaded)
	}

	// 3. Download non-existent & Delete
	if _, missingErr := client.Download(ctx, "nonexistent.png"); missingErr == nil {
		t.Error("expected error for nonexistent download, got nil")
	}
	if delErr := client.Delete(ctx, "test.png"); delErr != nil {
		t.Fatalf("delete failed: %v", delErr)
	}
}

func TestClient_PresignedAndEnsureBucket(t *testing.T) {
	t.Parallel()

	server := newMockS3Server(nil)
	defer server.Close()

	client, err := NewClient(&Config{
		Endpoint:  server.URL,
		Bucket:    mockBucket,
		AccessKey: mockAuth,
		SecretKey: mockAuth,
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	presignedURL, urlErr := client.GetPresignedURL(ctx, "test.png", 15*time.Minute)
	if urlErr != nil || !strings.Contains(presignedURL, mockBucket) {
		t.Fatalf("unexpected presigned url: %s (err: %v)", presignedURL, urlErr)
	}
	if ensureErr := client.EnsureBucket(ctx); ensureErr != nil {
		t.Fatalf("ensure bucket (existing) failed: %v", ensureErr)
	}

	newBucketClient, newErr := NewClient(&Config{
		Endpoint:  server.URL,
		Bucket:    "new-bucket",
		AccessKey: mockAuth,
		SecretKey: mockAuth,
	})
	if newErr != nil || newBucketClient.EnsureBucket(ctx) != nil {
		t.Fatalf("ensure bucket (new) failed: %v", newErr)
	}
}

func TestClient_Errors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>InternalError</Code><Message>Something went wrong</Message></Error>`))
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		Endpoint:  server.URL,
		Bucket:    "error-bucket",
		AccessKey: mockAuth,
		SecretKey: mockAuth,
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	if err := client.Upload(canceledCtx, "fail.png", bytes.NewReader([]byte("data")), 4, "image/png"); err == nil {
		t.Error("expected upload error, got nil")
	}

	if _, err := client.Download(ctx, "fail.png"); err == nil {
		t.Error("expected download error, got nil")
	}

	if err := client.EnsureBucket(canceledCtx); err == nil {
		t.Error("expected EnsureBucket error, got nil")
	}
}
