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
}
