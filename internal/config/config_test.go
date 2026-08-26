package config_test

import (
	"os"
	"testing"

	"planning-poker/internal/config"
)

func TestLoadTestConfig_UsesRedisEnvironmentOverrides(t *testing.T) {
	t.Setenv("REDIS_HOST", "redis.example.test")
	t.Setenv("REDIS_PORT", "6381")
	t.Setenv("REDIS_DB", "7")

	cfg, err := config.LoadTestConfig()
	if err != nil {
		t.Fatalf("load test config: %v", err)
	}

	if cfg.Redis.Host != "redis.example.test" {
		t.Fatalf("expected Redis host override, got %q", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6381 {
		t.Fatalf("expected Redis port override, got %d", cfg.Redis.Port)
	}
	if cfg.Redis.DB != 7 {
		t.Fatalf("expected Redis DB override, got %d", cfg.Redis.DB)
	}
}

func TestLoadTestConfig_RestoresConfigFileEnvironment(t *testing.T) {
	t.Setenv("CONFIG_FILE", "config.yaml")

	if _, err := config.LoadTestConfig(); err != nil {
		t.Fatalf("load test config: %v", err)
	}

	if got, ok := os.LookupEnv("CONFIG_FILE"); !ok || got != "config.yaml" {
		t.Fatalf("expected CONFIG_FILE to be restored, got %q", got)
	}
}
