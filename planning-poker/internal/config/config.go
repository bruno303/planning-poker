package config

import (
	"planning-poker/config"
	"time"

	toolkitconfig "github.com/bruno303/go-toolkit/pkg/config"
)

type Config struct {
	Service     string `env:"SERVICE" yaml:"service"`
	Environment string `env:"ENVIRONMENT" yaml:"environment"`
	API         struct {
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
		Admin struct {
			APIKey string `env:"ADMIN_API_KEY" yaml:"api_key"`
		} `yaml:"admin"`
	} `yaml:"api"`
	Trace struct {
		Enabled      bool   `env:"TRACE_ENABLED" yaml:"enabled"`
		OtlpEndpoint string `env:"TRACE_OTLP_ENDPOINT" yaml:"otlp_endpoint"`
	} `yaml:"trace"`
	LogLevel string `env:"LOG_LEVEL" yaml:"log_level"`
	Metrics  struct {
		Enabled bool   `env:"METRICS_ENABLED" yaml:"enabled"`
		Port    int    `env:"METRICS_PORT" yaml:"port"`
		Path    string `env:"METRICS_PATH" yaml:"path"`
	} `yaml:"metrics"`
	Redis struct {
		Host     string `env:"REDIS_HOST" yaml:"host"`
		Port     int    `env:"REDIS_PORT" yaml:"port"`
		Password string `env:"REDIS_PASSWORD" yaml:"password"`
		DB       int    `env:"REDIS_DB" yaml:"db"`
	} `yaml:"redis"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	toolkitconfig.LoadConfig(cfg, config.ConfigFS)
	return cfg, nil
}
