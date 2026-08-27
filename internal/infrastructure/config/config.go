package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string   `json:"port"`
	JWTSecret      string   `json:"jwt_secret"`
	DatabaseURL    string   `json:"database_url"`
	AllowedOrigins []string `json:"allowed_origins"`
}

var (
	ErrPortRequired           = errors.New("port is required and cannot be empty")
	ErrJWTSecretRequired      = errors.New("jwt_secret is required and cannot be empty")
	ErrDatabaseURLRequired    = errors.New("database_url is required and cannot be empty")
	ErrAllowedOriginsRequired = errors.New("allowed_origins is required and cannot be empty")
)

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config file %s is missing: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if envPort := os.Getenv("PORT"); envPort != "" {
		cfg.Port = envPort
	}
	if envJWTSecret := os.Getenv("JWT_SECRET"); envJWTSecret != "" {
		cfg.JWTSecret = envJWTSecret
	}
	if envDbURL := os.Getenv("DATABASE_URL"); envDbURL != "" {
		cfg.DatabaseURL = envDbURL
	}
	if envOrigins := os.Getenv("ALLOWED_ORIGINS"); envOrigins != "" {
		cfg.AllowedOrigins = parseOrigins(envOrigins)
	}
}

func parseOrigins(raw string) []string {
	var origins []string
	for o := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func (c *Config) Validate() error {
	if c.Port == "" {
		return ErrPortRequired
	}
	if c.JWTSecret == "" {
		return ErrJWTSecretRequired
	}
	if c.DatabaseURL == "" {
		return ErrDatabaseURLRequired
	}
	if len(c.AllowedOrigins) == 0 {
		return ErrAllowedOriginsRequired
	}
	return nil
}
