package config

import (
	"planning-poker/config"

	toolkitconfig "github.com/bruno303/go-toolkit/pkg/config"
)

type Config struct {
	API struct {
		BackendPort        int    `env:"API_BACKEND_PORT" yaml:"backend_port"`
		CorsAllowedOrigins string `env:"API_CORS_ALLOWED_ORIGINS" yaml:"cors_allowed_origins"`
	} `yaml:"api"`
	TraceOtlpEndpoint string `env:"TRACE_OTLP_ENDPOINT" yaml:"trace_otlp_endpoint"`
	LogLevel          string `env:"LOG_LEVEL" yaml:"log_level"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	toolkitconfig.LoadConfig(cfg, config.ConfigFS)
	return cfg, nil
}
