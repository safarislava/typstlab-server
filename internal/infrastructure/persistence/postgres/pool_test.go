package postgres

import (
	"context"
	"testing"
)

const testValidDatabaseURL = "postgres://user:pass@localhost:5432/testdb?sslmode=disable"

func TestNewPool_InvalidURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	invalidURL := "invalid://not a valid url %%%"

	pool, err := NewPool(ctx, invalidURL)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected error for invalid database URL, got nil")
	}
}

func TestNewPool_ValidURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("unexpected error creating pool with valid URL: %v", err)
	}
	defer pool.Close()

	config := pool.Config()
	if config.MaxConns != defaultMaxConns {
		t.Errorf("expected MaxConns %d, got %d", defaultMaxConns, config.MaxConns)
	}
	if config.MinConns != defaultMinConns {
		t.Errorf("expected MinConns %d, got %d", defaultMinConns, config.MinConns)
	}
	if config.MaxConnLifetime != defaultMaxConnLifetime {
		t.Errorf("expected MaxConnLifetime %v, got %v", defaultMaxConnLifetime, config.MaxConnLifetime)
	}
	if config.MaxConnIdleTime != defaultMaxConnIdleTime {
		t.Errorf("expected MaxConnIdleTime %v, got %v", defaultMaxConnIdleTime, config.MaxConnIdleTime)
	}
	if config.HealthCheckPeriod != defaultHealthCheckPeriod {
		t.Errorf("expected HealthCheckPeriod %v, got %v", defaultHealthCheckPeriod, config.HealthCheckPeriod)
	}
}
