package config

import (
	"testing"
)

const (
	testJWTSecret   = "secret"
	testDatabaseURL = "postgres://localhost:5432/test"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Load panicked: %v", r)
		}
	}()

	cfg := Load("../../../configs/config.json")
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.Port == "" {
		t.Error("Expected port to be non-empty")
	}

	if cfg.JWTSecret == "" {
		t.Error("Expected jwt_secret to be non-empty")
	}

	if cfg.DatabaseURL == "" {
		t.Error("Expected database_url to be non-empty")
	}
	if len(cfg.AllowedOrigins) == 0 {
		t.Error("Expected allowed_origins to be non-empty")
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	validConfig := Config{
		Port:        "8080",
		JWTSecret:   testJWTSecret,
		DatabaseURL: testDatabaseURL,
	}
	if err := validConfig.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	invalidConfig1 := Config{
		Port:        "",
		JWTSecret:   testJWTSecret,
		DatabaseURL: testDatabaseURL,
	}
	if err := invalidConfig1.Validate(); err == nil {
		t.Error("Expected error for empty Port, got nil")
	}

	invalidConfig2 := Config{
		Port:        "8080",
		JWTSecret:   "",
		DatabaseURL: testDatabaseURL,
	}
	if err := invalidConfig2.Validate(); err == nil {
		t.Error("Expected error for empty JWT secret, got nil")
	}

	invalidConfig3 := Config{
		Port:        "8080",
		JWTSecret:   testJWTSecret,
		DatabaseURL: "",
	}
	if err := invalidConfig3.Validate(); err == nil {
		t.Error("Expected error for empty DatabaseURL, got nil")
	}
}

//nolint:paralleltest // t.Setenv cannot be used in parallel tests
func TestConfig_EnvOverride(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("ALLOWED_ORIGINS", "https://example.com, https://test.com")

	cfg := Load("../../../configs/config.json")
	if cfg.Port != "9999" {
		t.Errorf("Expected PORT env override to be 9999, got %s", cfg.Port)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "https://example.com" || cfg.AllowedOrigins[1] != "https://test.com" {
		t.Errorf("Expected ALLOWED_ORIGINS env override to be [https://example.com, https://test.com], got %v", cfg.AllowedOrigins)
	}
}
