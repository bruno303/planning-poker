package main

import (
	"context"
	"net/http"
	"planning-poker/internal/infra"
	"planning-poker/internal/planningpoker"

	"github.com/bruno303/go-toolkit/pkg/log"
)

func main() {
	log.SetLoggerFactory(func(name string) log.Logger {
		return log.NewSlogAdapter(
			log.SlogAdapterOpts{
				Level:                 log.LevelInfo,
				FormatJson:            false,
				ExtractAdditionalInfo: func(context.Context) []any { return []any{} },
				Source:                name,
			},
		)
	})
	container := NewContainer()

	ctx := context.Background()
	logger := log.NewLogger("main")

	mux := http.NewServeMux()
	planningpoker.ConfigurePlanningPokerAPI(mux, container.Hub)
	infra.ConfigureInfraAPI(mux)

	logger.Info(ctx, "Starting server on :8080")
	http.ListenAndServe(":8080", mux)
}
