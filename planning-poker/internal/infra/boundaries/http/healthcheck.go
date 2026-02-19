package http

import (
	"context"
	"net/http"
	"time"
)

type (
	HealthChecker interface {
		Name() string
		Check(ctx context.Context) HealthStatus
	}

	HealthStatus struct {
		Status  string         `json:"status"`
		Message string         `json:"message,omitempty"`
		Details map[string]any `json:"details,omitempty"`
	}

	HealthcheckAPI struct {
		checkers []HealthChecker
		timeout  time.Duration
	}

	HealthcheckResponse struct {
		Status    string                  `json:"status"`
		Timestamp int64                   `json:"timestamp"`
		Checks    map[string]HealthStatus `json:"checks"`
		Summary   map[string]int          `json:"summary"`
	}
)

const (
	HealthStatusPass = "ok"
	HealthStatusFail = "fail"
	HealthStatusWarn = "warn"
)

var _ API = (*HealthcheckAPI)(nil)

func NewHealthcheckAPI(checkers ...HealthChecker) HealthcheckAPI {
	return HealthcheckAPI{
		checkers: checkers,
		timeout:  5 * time.Second,
	}
}

func (api HealthcheckAPI) Endpoint() string {
	return "/health"
}

func (api HealthcheckAPI) Methods() []string {
	return []string{"GET", "HEAD"}
}

func (api HealthcheckAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), api.timeout)
		defer cancel()

		response := api.performHealthChecks(ctx)

		statusCode := http.StatusOK
		if response.Status == HealthStatusFail {
			statusCode = http.StatusServiceUnavailable
		}

		SendJsonResponse(w, statusCode, response)
	})
}

func (api HealthcheckAPI) performHealthChecks(ctx context.Context) HealthcheckResponse {
	checks := make(map[string]HealthStatus)
	summary := map[string]int{
		HealthStatusPass: 0,
		HealthStatusFail: 0,
		HealthStatusWarn: 0,
	}

	for _, checker := range api.checkers {
		checkCtx, checkCancel := context.WithTimeout(ctx, api.timeout)
		status := checker.Check(checkCtx)
		checkCancel()

		checks[checker.Name()] = status
		summary[status.Status]++
	}

	overallStatus := HealthStatusPass
	if summary[HealthStatusFail] > 0 {
		overallStatus = HealthStatusFail
	} else if summary[HealthStatusWarn] > 0 {
		overallStatus = HealthStatusWarn
	}

	return HealthcheckResponse{
		Status:    overallStatus,
		Timestamp: time.Now().Unix(),
		Checks:    checks,
		Summary:   summary,
	}
}
