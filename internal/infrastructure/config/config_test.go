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
		{"valid", Config{"8080", testJWTSecret, testDatabaseURL, origins}, nil},
		{"missing port", Config{"", testJWTSecret, testDatabaseURL, origins}, ErrPortRequired},
		{"missing jwt", Config{"8080", "", testDatabaseURL, origins}, ErrJWTSecretRequired},
		{"missing db url", Config{"8080", testJWTSecret, "", origins}, ErrDatabaseURLRequired},
		{"nil origins", Config{"8080", testJWTSecret, testDatabaseURL, nil}, ErrAllowedOriginsRequired},
		{"empty origins", Config{"8080", testJWTSecret, testDatabaseURL, []string{}}, ErrAllowedOriginsRequired},
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
}
