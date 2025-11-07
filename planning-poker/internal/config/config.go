package config

import (
	"planning-poker/config"
	"time"

	toolkitconfig "github.com/bruno303/go-toolkit/pkg/config"
)

type Config struct {
	Service string `env:"SERVICE" yaml:"service"`
	API     struct {
		Tracing struct {
			Enabled bool `env:"API_TRACING_ENABLED" yaml:"enabled"`
		} `yaml:"tracing"`
		BackendPort        int    `env:"API_BACKEND_PORT" yaml:"backend_port"`
		CorsAllowedOrigins string `env:"API_CORS_ALLOWED_ORIGINS" yaml:"cors_allowed_origins"`
		PlanningPoker      struct {
			WebsocketWriteTimeout time.Duration `env:"API_PLANNING_POKER_WEBSOCKET_WRITE_TIMEOUT" yaml:"websocket_write_timeout"`
			WebsocketReadTimeout  time.Duration `env:"API_PLANNING_POKER_WEBSOCKET_READ_TIMEOUT" yaml:"websocket_read_timeout"`
			WebsocketPingInterval time.Duration `env:"API_PLANNING_POKER_WEBSOCKET_PING_INTERVAL" yaml:"websocket_ping_interval"`
		} `yaml:"planning_poker"`
	} `yaml:"api"`
	TraceOtlpEndpoint string `env:"TRACE_OTLP_ENDPOINT" yaml:"trace_otlp_endpoint"`
	LogLevel          string `env:"LOG_LEVEL" yaml:"log_level"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	toolkitconfig.LoadConfig(cfg, config.ConfigFS)
	return cfg, nil
}
