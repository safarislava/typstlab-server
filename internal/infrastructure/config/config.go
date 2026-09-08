package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type S3Config struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	UseSSL    bool   `json:"use_ssl"`
	Region    string `json:"region"`
}

type Config struct {
	Port           string   `json:"port"`
	JWTSecret      string   `json:"jwt_secret"`
	DatabaseURL    string   `json:"database_url"`
	AllowedOrigins []string `json:"allowed_origins"`
	S3             S3Config `json:"s3"`
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
	applyBaseEnvOverrides(cfg)
	applyS3EnvOverrides(&cfg.S3)
}

func applyBaseEnvOverrides(cfg *Config) {
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

func applyS3EnvOverrides(s3Cfg *S3Config) {
	if envS3Endpoint := os.Getenv("S3_ENDPOINT"); envS3Endpoint != "" {
		s3Cfg.Endpoint = envS3Endpoint
	}
	if envS3Bucket := os.Getenv("S3_BUCKET"); envS3Bucket != "" {
		s3Cfg.Bucket = envS3Bucket
	}
	if envS3AccessKey := os.Getenv("S3_ACCESS_KEY"); envS3AccessKey != "" {
		s3Cfg.AccessKey = envS3AccessKey
	}
	if envS3SecretKey := os.Getenv("S3_SECRET_KEY"); envS3SecretKey != "" {
		s3Cfg.SecretKey = envS3SecretKey
	}
	if envS3UseSSL := os.Getenv("S3_USE_SSL"); envS3UseSSL != "" {
		if val, err := strconv.ParseBool(envS3UseSSL); err == nil {
			s3Cfg.UseSSL = val
		}
	}
	if envS3Region := os.Getenv("S3_REGION"); envS3Region != "" {
		s3Cfg.Region = envS3Region
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
