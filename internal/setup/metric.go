package setup

import (
	"context"
	"planning-poker/internal/config"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/metric"
)

func ConfigureMetrics(ctx context.Context, cfg *config.Config, logger log.Logger) (func(context.Context) error, error) {
	return metric.SetupOTelMetrics(ctx, metric.Config{
		ApplicationName:    cfg.Service,
		ApplicationVersion: "0.0.1",
		Environment:        cfg.Environment,
		Enabled:            cfg.Metrics.Enabled,
		Port:               cfg.Metrics.Port,
		Path:               cfg.Metrics.Path,
		Log:                logger,
	})
}
