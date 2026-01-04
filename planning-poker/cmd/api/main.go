package main

import (
	"context"
	"fmt"
	"net/http"
	"planning-poker/internal/config"
	"planning-poker/internal/setup"
	"strings"

	"github.com/bruno303/go-toolkit/pkg/log"
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

	setup.ConfigureLogging(cfg)
	logger := log.NewLogger("main")

	metricsShutdown, err := setup.ConfigureMetrics(ctx, cfg, logger)
	if err != nil {
		logger.Error(ctx, "Error configuring metrics", err)
		return
	}
	defer func() {
		if err := metricsShutdown(ctx); err != nil {
			logger.Error(ctx, "Error shutting down metrics", err)
		}
	}()

	shutdown := setup.ConfigureTrace(ctx, cfg, logger)
	defer func() {
		if err := shutdown(ctx); err != nil {
			logger.Error(ctx, "Error shutting down tracing", err)
		}
	}()

	container := setup.NewContainer(cfg)

	r := mux.NewRouter()
	configureMiddlewares(ctx, r, logger)
	setup.ConfigureAPIs(r, container)

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

func loggingMiddleware(next http.Handler, logger log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.Context(), "Request: %s %s from Origin: %s", r.Method, r.URL.Path, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
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
