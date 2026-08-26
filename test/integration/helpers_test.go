package integration

import (
	"testing"
)

func TestRedisOptions_UsesEnvironmentHostAndPort(t *testing.T) {
	t.Setenv("REDIS_HOST", "redis.example.test")
	t.Setenv("REDIS_PORT", "6381")

	options := RedisOptions(t, 5)
	if options.Addr != "redis.example.test:6381" {
		t.Fatalf("expected custom Redis address, got %q", options.Addr)
	}
	if options.DB != 5 {
		t.Fatalf("expected default database 5, got %d", options.DB)
	}
}

func TestRedisOptions_UsesLocalDefaults(t *testing.T) {
	t.Setenv("REDIS_HOST", "")
	t.Setenv("REDIS_PORT", "")

	options := RedisOptions(t, 6)
	if options.Addr != "localhost:6379" {
		t.Fatalf("expected local Redis address, got %q", options.Addr)
	}
	if options.DB != 6 {
		t.Fatalf("expected default database 6, got %d", options.DB)
	}
}
