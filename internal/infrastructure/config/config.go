package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string   `json:"port"`
	JWTSecret      string   `json:"jwt_secret"`
	DatabaseURL    string   `json:"database_url"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
}

func Load(path string) *Config {
	file, err := os.Open(path)
	if err != nil {
		panic("config file " + path + " is missing: " + err.Error())
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		panic("failed to decode config: " + err.Error())
	}

	applyEnvOverrides(&cfg)

	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"https://*", "http://*"}
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if err := cfg.Validate(); err != nil {
		panic("invalid configuration: " + err.Error())
	}

	return &cfg
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
		return fmt.Errorf("port is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required and cannot be empty")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required and cannot be empty")
	}
	return nil
}
