package main

import (
	"context"
	"fmt"
	"net/http"
	"planning-poker/internal/config"
	"planning-poker/internal/infra"
	httpapp "planning-poker/internal/infra/boundaries/http"
	"strings"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

var cfg *config.Config

func main() {
	ctx := context.Background()

	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		panic(err)
	}

	configureLogging()
	logger := log.NewLogger("main")
	shutdown := configureTrace(ctx, logger)
	defer func() {
		if err := shutdown(ctx); err != nil {
			logger.Error(ctx, "Error shutting down tracing", err)
		}
	}()

	container := NewContainer()

	r := mux.NewRouter()
	configureMiddlewares(ctx, r, logger)
	httpapp.ConfigurePlanningPokerAPI(r, container.Hub, container.Usecases, httpapp.WebSocketConfig{
		WriteTimeout: cfg.API.PlanningPoker.WebsocketWriteTimeout,
		ReadTimeout:  cfg.API.PlanningPoker.WebsocketReadTimeout,
		PingInterval: cfg.API.PlanningPoker.WebsocketPingInterval,
	})
	infra.ConfigureInfraAPI(r)

	walkRoutes(ctx, r, logger)
	port := cfg.API.BackendPort
	if port == 0 {
		port = 8080
	}

	logger.Info(ctx, "Listening on port %d", port)
	if err := http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		loggingMiddleware(corsMiddleware(r, logger), logger),
	); err != nil {
		logger.Error(ctx, "Error starting server", err)
	}
}

func configureMiddlewares(ctx context.Context, r *mux.Router, logger log.Logger) {
	if cfg.API.Tracing.Enabled {
		logger.Info(ctx, "Tracing on API enabled")
		r.Use(otelmux.Middleware(cfg.Service))
	} else {
		logger.Info(ctx, "Tracing on API disabled")
	}
}

func configureLogging() {
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

	logFactory := func(name string) log.Logger {
		return log.NewSlogAdapter(
			log.SlogAdapterOpts{
				Level:                 ll,
				FormatJson:            false,
				ExtractAdditionalInfo: func(context.Context) []any { return []any{} },
				Name:                  name,
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

func getCORSOrigins(logger log.Logger) []string {
	var allowedOrigins []string
	corsOrigins := cfg.API.CorsAllowedOrigins

	if corsOrigins == "" {
		allowedOrigins = []string{
			"*",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:8080",
		}
	} else {
		allowedOrigins = strings.Split(corsOrigins, ",")
	}

	logger.Debug(context.Background(), "Allowed origins: %v", allowedOrigins)
	return allowedOrigins
}

func loggingMiddleware(next http.Handler, logger log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.Context(), "Request: %s %s from Origin: %s", r.Method, r.URL.Path, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler, logger log.Logger) http.Handler {
	middleware := handlers.CORS(
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS", "PUT", "DELETE"}),
		handlers.AllowedOrigins(getCORSOrigins(logger)),
		handlers.AllowedHeaders([]string{"content-type", "authorization", "origin"}),
		handlers.AllowCredentials(),
		handlers.OptionStatusCode(http.StatusOK),
	)
	return middleware(next)
}

func configureTrace(ctx context.Context, logger log.Logger) func(context.Context) error {
	shutdown, err := trace.SetupOTelSDK(ctx, trace.Config{
		ApplicationName:    "planning-poker-backend",
		ApplicationVersion: "0.0.1",
		Endpoint:           cfg.TraceOtlpEndpoint,
	})
	if err != nil {
		logger.Error(ctx, "Error setting up tracing: %v", err)
	} else {
		trace.SetTracer(trace.NewOtelTracerAdapter())
	}
	return shutdown
}

func walkRoutes(ctx context.Context, r *mux.Router, logger log.Logger) {
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			logger.Debug(ctx, "ROUTE: %v", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			logger.Debug(ctx, "Path regexp: %v", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			logger.Debug(ctx, "Queries templates: %v", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			logger.Debug(ctx, "Queries regexps: %v", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			logger.Debug(ctx, "Methods: %v", strings.Join(methods, ","))
		}
		logger.Debug(ctx, "")
		return nil
	})

	if err != nil {
		logger.Error(ctx, "Error walking routes", err)
	}
}
