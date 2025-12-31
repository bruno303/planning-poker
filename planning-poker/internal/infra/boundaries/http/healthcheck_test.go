package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckAPI_Endpoint(t *testing.T) {
	api := NewHealthcheckAPI()

	if api.Endpoint() != "/health" {
		t.Errorf("Endpoint() = %v, want %v", api.Endpoint(), "/health")
	}
}

func TestHealthcheckAPI_Methods(t *testing.T) {
	api := NewHealthcheckAPI()
	methods := api.Methods()

	if len(methods) != 1 {
		t.Fatalf("Methods() length = %v, want %v", len(methods), 1)
	}
	if methods[0] != "GET" {
		t.Errorf("Methods()[0] = %v, want %v", methods[0], "GET")
	}
}

func TestHealthcheckAPI_Handle_Success(t *testing.T) {
	api := NewHealthcheckAPI()
	handler := api.Handle()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	var response HealthcheckResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Status = %v, want %v", response.Status, "ok")
	}
}

func TestHealthcheckAPI_Handle_DifferentMethods(t *testing.T) {
	api := NewHealthcheckAPI()
	handler := api.Handle()

	// Although the API specifies GET, the handler should respond to any method
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
			}

			var response HealthcheckResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Status != "ok" {
				t.Errorf("Status = %v, want %v", response.Status, "ok")
			}
		})
	}
}
