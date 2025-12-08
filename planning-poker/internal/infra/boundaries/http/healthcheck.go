package http

import (
	"net/http"
)

type (
	HealthcheckAPI      struct{}
	HealthcheckResponse struct {
		Status string `json:"status"`
	}
)

var _ API = (*HealthcheckAPI)(nil)

func NewHealthcheckAPI() HealthcheckAPI {
	return HealthcheckAPI{}
}

func (api HealthcheckAPI) Endpoint() string {
	return "/health"
}

func (api HealthcheckAPI) Methods() []string {
	return []string{"GET"}
}

func (api HealthcheckAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SendJsonResponse(w, http.StatusOK, HealthcheckResponse{Status: "ok"})
	})
}
