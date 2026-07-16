package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port      string `json:"port"`
	JWTSecret string `json:"jwt_secret"`
}

func Load() *Config {
	file, err := os.Open("configs/config.json")
	if err != nil {
		panic("config file configs/config.json is missing: " + err.Error())
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		panic("failed to decode config: " + err.Error())
	}

	return &cfg
}
