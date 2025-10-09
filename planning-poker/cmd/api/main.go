package main

import (
	"context"
	"net/http"
	"os"
	"planning-poker/internal/infra"
	httpapp "planning-poker/internal/infra/boundaries/http"
	"strings"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	ctx := context.Background()

	configureLogging()
	logger := log.NewLogger("main")

	container := NewContainer()

	r := mux.NewRouter()
	httpapp.ConfigurePlanningPokerAPI(r, container.Hub, container.Usecases)
	infra.ConfigureInfraAPI(r)

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

	logger.Info(ctx, "Starting server on :8080")
	if err = http.ListenAndServe(
		":8080",
		loggingMiddleware(corsMiddleware(r, logger), logger),
	); err != nil {
		logger.Error(ctx, "Error starting server", err)
	}
}

func configureLogging() {
	logLevel := os.Getenv("LOG_LEVEL")

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

	log.SetLoggerFactory(func(name string) log.Logger {
		return log.NewSlogAdapter(
			log.SlogAdapterOpts{
				Level:                 ll,
				FormatJson:            false,
				ExtractAdditionalInfo: func(context.Context) []any { return []any{} },
				Source:                name,
			},
		)
	})
}

func getCORSOrigins(logger log.Logger) []string {
	var allowedOrigins []string
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")

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
