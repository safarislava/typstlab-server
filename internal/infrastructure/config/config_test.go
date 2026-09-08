package config

import (
	"errors"
	"testing"
)

const (
	testJWTSecret   = "secret"
	testDatabaseURL = "postgres://localhost:5432/test"
	testOrigin      = "http://localhost:3000"
)

func TestLoad_Success(t *testing.T) {
	t.Parallel()

	cfg, err := Load("../../../configs/config.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.Port == "" || cfg.JWTSecret == "" || cfg.DatabaseURL == "" || len(cfg.AllowedOrigins) == 0 {
		t.Error("Expected all config fields to be populated")
	}

	if cfg.S3.Endpoint == "" || cfg.S3.Bucket == "" {
		t.Error("Expected S3 config fields to be populated")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := Load("non_existent_file.json")
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	origins := []string{testOrigin}
	tests := []struct {
		name        string
		cfg         Config
		expectedErr error
	}{
		{"valid", Config{Port: "8080", JWTSecret: testJWTSecret, DatabaseURL: testDatabaseURL, AllowedOrigins: origins}, nil},
		{"missing port", Config{Port: "", JWTSecret: testJWTSecret, DatabaseURL: testDatabaseURL, AllowedOrigins: origins}, ErrPortRequired},
		{"missing jwt", Config{Port: "8080", JWTSecret: "", DatabaseURL: testDatabaseURL, AllowedOrigins: origins}, ErrJWTSecretRequired},
		{"missing db url", Config{Port: "8080", JWTSecret: testJWTSecret, DatabaseURL: "", AllowedOrigins: origins}, ErrDatabaseURLRequired},
		{"nil origins", Config{Port: "8080", JWTSecret: testJWTSecret, DatabaseURL: testDatabaseURL, AllowedOrigins: nil}, ErrAllowedOriginsRequired},
		{"empty origins", Config{Port: "8080", JWTSecret: testJWTSecret, DatabaseURL: testDatabaseURL, AllowedOrigins: []string{}}, ErrAllowedOriginsRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

//nolint:paralleltest // t.Setenv cannot be used in parallel tests
func TestConfig_EnvOverride(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("ALLOWED_ORIGINS", "https://example.com, https://test.com")
	t.Setenv("S3_ENDPOINT", "s3.custom.com")
	t.Setenv("S3_BUCKET", "custom-bucket")
	t.Setenv("S3_ACCESS_KEY", "custom-key")
	t.Setenv("S3_SECRET_KEY", "custom-secret")
	t.Setenv("S3_USE_SSL", "true")
	t.Setenv("S3_REGION", "eu-central-1")

	cfg, err := Load("../../../configs/config.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Port != "9999" {
		t.Errorf("Expected PORT env override to be 9999, got %s", cfg.Port)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "https://example.com" || cfg.AllowedOrigins[1] != "https://test.com" {
		t.Errorf("Expected ALLOWED_ORIGINS env override to be [https://example.com, https://test.com], got %v", cfg.AllowedOrigins)
	}
	expectedS3 := S3Config{
		Endpoint:  "s3.custom.com",
		Bucket:    "custom-bucket",
		AccessKey: "custom-key",
		SecretKey: "custom-secret",
		UseSSL:    true,
		Region:    "eu-central-1",
	}
	if cfg.S3 != expectedS3 {
		t.Errorf("Expected S3 env overrides to match %+v, got %+v", expectedS3, cfg.S3)
	}
}
