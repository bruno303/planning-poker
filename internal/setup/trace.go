package setup

import (
	"context"
	"planning-poker/internal/config"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
)

func ConfigureTrace(ctx context.Context, cfg *config.Config, logger log.Logger) func(context.Context) error {
	if !cfg.Trace.Enabled {
		logger.Info(ctx, "Tracing disabled")
		return func(context.Context) error { return nil }
	}

	shutdown, err := trace.SetupOTelSDK(ctx, trace.Config{
		ApplicationName:    cfg.Service,
		ApplicationVersion: "0.0.1",
		Endpoint:           cfg.Trace.OtlpEndpoint,
		Environment:        cfg.Environment,
	})
	if err != nil {
		logger.Error(ctx, "Error setting up tracing: %v", err)
	} else {
		trace.SetTracer(trace.NewOtelTracerAdapter())
	}
	return shutdown
}
