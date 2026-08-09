package config

import (
	"testing"
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
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	validConfig := Config{
		Port:      "8080",
		JWTSecret: "secret",
	}
	if err := validConfig.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	invalidConfig := Config{
		Port:      "8080",
		JWTSecret: "",
	}
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected error for empty JWT secret, got nil")
	}
}

//nolint:paralleltest // t.Setenv cannot be used in parallel tests
func TestConfig_EnvOverride(t *testing.T) {
	t.Setenv("PORT", "9999")

	cfg := Load("../../../configs/config.json")
	if cfg.Port != "9999" {
		t.Errorf("Expected PORT env override to be 9999, got %s", cfg.Port)
	}
}
