package setup

import (
	"context"
	"planning-poker/internal/config"
	"strings"

	"github.com/bruno303/go-toolkit/pkg/log"
)

func ConfigureLogging(cfg *config.Config) {
	logLevel := cfg.LogLevel

	var ll log.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		ll = log.LevelDebug
	case "INFO":
		ll = log.LevelInfo
	case "WARN":
		ll = log.LevelWarn
	case "ERROR":
		ll = log.LevelError
	default:
		ll = log.LevelInfo
	}

	var jsonEnvironments = map[string]bool{
		"production":  true,
		"staging":     true,
		"development": true,
	}
	logFactory := func(name string) log.Logger {
		formatJson := jsonEnvironments[cfg.Environment]

		return log.NewSlogAdapter(
			log.SlogAdapterOpts{
				Level:                 ll,
				FormatJson:            formatJson,
				ExtractAdditionalInfo: func(context.Context) []any { return []any{} },
				Name:                  name,
				Environment:           cfg.Environment,
			},
		)
	}

	log.ConfigureLogging(log.LogConfig{
		Type: log.LogTypeMultiple,
		MultipleLogConfig: log.MultipleLogConfig{
			Factory: logFactory,
		},
	})
}
