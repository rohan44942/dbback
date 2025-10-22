package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type StorageConfig struct {
	Type      string `yaml:"type"` // local | s3
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint"` //  for S3-compatible (MinIO)
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type AppConfig struct {
	DBPath    string        `yaml:"db_path"` // sqlite metadata path
	Storage   StorageConfig `yaml:"storage"`
	Retention int           `yaml:"retention_days"`
	Key       string        `yaml:"encryption_key"` //  32-byte key
	Server    ServerConfig  `yaml:"server"`
}

func Load(path string) (*AppConfig, error) {
	cfg := &AppConfig{}
	// defaults
	cfg.DBPath = "metadata/metadata.db"
	cfg.Retention = 7
	cfg.Server.Addr = ":8080"

	data, err := os.ReadFile(path)
	if err != nil {
		// fallback to env-only if file missing
		// read envs into cfg
		loadFromEnv(cfg)
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// overlay env if present
	loadFromEnv(cfg)
	return cfg, nil
}

func loadFromEnv(cfg *AppConfig) {
	if v := os.Getenv("DBBACK_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("DBBACK_RETENTION_DAYS"); v != "" {
		// ignore parse error for brevity
		// prefer YAML; envs override if present
	}
	if v := os.Getenv("DBBACK_STORAGE_TYPE"); v != "" {
		cfg.Storage.Type = v
	}
	if v := os.Getenv("DBBACK_STORAGE_BUCKET"); v != "" {
		cfg.Storage.Bucket = v
	}
	if v := os.Getenv("DBBACK_STORAGE_ENDPOINT"); v != "" {
		cfg.Storage.Endpoint = v
	}
	if v := os.Getenv("DBBACK_STORAGE_ACCESS_KEY"); v != "" {
		cfg.Storage.AccessKey = v
	}
	if v := os.Getenv("DBBACK_STORAGE_SECRET_KEY"); v != "" {
		cfg.Storage.SecretKey = v
	}
	if v := os.Getenv("DBBACK_KEY"); v != "" {
		cfg.Key = v
	}
	if v := os.Getenv("DBBACK_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
}
