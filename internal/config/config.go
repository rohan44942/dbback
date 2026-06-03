package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type StorageConfig struct {
	Type      string `yaml:"type"`
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type AppConfig struct {
	DatabaseURL     string        `yaml:"database_url"`
	Storage         StorageConfig `yaml:"storage"`
	Retention       int           `yaml:"retention_days"`
	Key             string        `yaml:"encryption_key"`
	Server          ServerConfig  `yaml:"server"`
	CORSOrigin      string        `yaml:"-"`
	SlackWebhookURL string        `yaml:"-"`
	Env             string        `yaml:"-"`
}

const defaultDevAuthSecret = "development-auth-secret-change-before-production"

func Load(path string) (*AppConfig, error) {
	cfg := &AppConfig{}
	cfg.DatabaseURL = "postgres://dbback:dbback@localhost:5432/dbback?sslmode=disable"
	cfg.Retention = 7
	cfg.Server.Addr = ":8080"

	data, err := os.ReadFile(path)
	if err != nil {
		loadFromEnv(cfg)
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	loadFromEnv(cfg)
	return cfg, nil
}

func loadFromEnv(cfg *AppConfig) {
	if v := os.Getenv("DBBACK_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("DBBACK_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Retention = n
		}
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
	if v := os.Getenv("DBBACK_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("DBBACK_CORS_ORIGIN"); v != "" {
		cfg.CORSOrigin = v
	}
	if v := os.Getenv("SLACK_WEBHOOK_URL"); v != "" {
		cfg.SlackWebhookURL = v
	}
	if v := os.Getenv("DBBACK_ENV"); v != "" {
		cfg.Env = v
	}
}

func AuthSecret() string {
	if secret := strings.TrimSpace(os.Getenv("DBBACK_AUTH_SECRET")); secret != "" {
		return secret
	}
	return defaultDevAuthSecret
}

func ValidateProduction(cfg *AppConfig) error {
	if strings.ToLower(cfg.Env) != "production" {
		return nil
	}
	if AuthSecret() == defaultDevAuthSecret {
		return errProductionAuthSecret
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return errProductionDatabaseURL
	}
	return nil
}

var (
	errProductionAuthSecret  = &configError{"DBBACK_AUTH_SECRET must be set in production"}
	errProductionDatabaseURL = &configError{"DBBACK_DATABASE_URL must be set in production"}
)

type configError struct{ msg string }

func (e *configError) Error() string { return e.msg }

func CORSOrigins(cfg *AppConfig) []string {
	raw := cfg.CORSOrigin
	if raw == "" {
		raw = "http://localhost:3000"
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
