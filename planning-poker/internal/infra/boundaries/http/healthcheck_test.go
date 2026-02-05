package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockHealthChecker struct {
	name   string
	status HealthStatus
}

func (m *MockHealthChecker) Name() string {
	return m.name
}

func (m *MockHealthChecker) Check(ctx context.Context) HealthStatus {
	return m.status
}

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

func TestHealthcheckAPI_Handle_NoCheckers(t *testing.T) {
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

	if response.Status != HealthStatusPass {
		t.Errorf("Status = %v, want %v", response.Status, HealthStatusPass)
	}
}

func TestHealthcheckAPI_PerformHealthChecks(t *testing.T) {
	tests := []struct {
		name     string
		checkers []HealthChecker
		expected HealthcheckResponse
	}{
		{
			name: "all checks pass",
			checkers: []HealthChecker{
				&MockHealthChecker{
					name: "test1",
					status: HealthStatus{
						Status:  HealthStatusPass,
						Message: "OK",
					},
				},
				&MockHealthChecker{
					name: "test2",
					status: HealthStatus{
						Status:  HealthStatusPass,
						Message: "OK",
					},
				},
			},
			expected: HealthcheckResponse{
				Status: HealthStatusPass,
				Summary: map[string]int{
					HealthStatusPass: 2,
					HealthStatusFail: 0,
					HealthStatusWarn: 0,
				},
			},
		},
		{
			name: "one check fails",
			checkers: []HealthChecker{
				&MockHealthChecker{
					name: "test1",
					status: HealthStatus{
						Status:  HealthStatusPass,
						Message: "OK",
					},
				},
				&MockHealthChecker{
					name: "test2",
					status: HealthStatus{
						Status:  HealthStatusFail,
						Message: "Error",
					},
				},
			},
			expected: HealthcheckResponse{
				Status: HealthStatusFail,
				Summary: map[string]int{
					HealthStatusPass: 1,
					HealthStatusFail: 1,
					HealthStatusWarn: 0,
				},
			},
		},
		{
			name: "one check warns",
			checkers: []HealthChecker{
				&MockHealthChecker{
					name: "test1",
					status: HealthStatus{
						Status:  HealthStatusPass,
						Message: "OK",
					},
				},
				&MockHealthChecker{
					name: "test2",
					status: HealthStatus{
						Status:  HealthStatusWarn,
						Message: "Warning",
					},
				},
			},
			expected: HealthcheckResponse{
				Status: HealthStatusWarn,
				Summary: map[string]int{
					HealthStatusPass: 1,
					HealthStatusFail: 0,
					HealthStatusWarn: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := NewHealthcheckAPI(tt.checkers...)

			ctx := context.Background()
			response := api.performHealthChecks(ctx)

			assert.Equal(t, tt.expected.Status, response.Status)
			assert.Equal(t, tt.expected.Summary, response.Summary)
			assert.Len(t, response.Checks, len(tt.checkers))

			for _, checker := range tt.checkers {
				check, exists := response.Checks[checker.Name()]
				require.True(t, exists)
				expectedChecker := checker.(*MockHealthChecker)
				assert.Equal(t, expectedChecker.status.Status, check.Status)
				assert.Equal(t, expectedChecker.status.Message, check.Message)
			}

			assert.Greater(t, response.Timestamp, int64(0))
		})
	}
}

func TestHealthcheckAPI_Handle_ServiceUnavailable(t *testing.T) {
	checkers := []HealthChecker{
		&MockHealthChecker{
			name: "test1",
			status: HealthStatus{
				Status:  HealthStatusFail,
				Message: "Error",
			},
		},
	}

	api := NewHealthcheckAPI(checkers...)
	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	api.Handle().ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestHealthEndpoints_Integration(t *testing.T) {
	// Create mock health checkers
	checkers := []HealthChecker{
		&MockHealthChecker{
			name: "redis",
			status: HealthStatus{
				Status:  HealthStatusPass,
				Message: "Redis connection successful",
				Details: map[string]any{"response_time_ms": 5},
			},
		},
		&MockHealthChecker{
			name: "websocket",
			status: HealthStatus{
				Status:  HealthStatusPass,
				Message: "WebSocket endpoint available",
			},
		},
		&MockHealthChecker{
			name: "metrics",
			status: HealthStatus{
				Status:  HealthStatusFail,
				Message: "Metrics collection failed",
				Details: map[string]any{"error": "port already in use"},
			},
		},
	}

	// Test unified health endpoint returns service unavailable when any component fails
	t.Run("unified health endpoint with failure", func(t *testing.T) {
		api := NewHealthcheckAPI(checkers...)
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		api.Handle().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		body, err := io.ReadAll(rec.Body)
		require.NoError(t, err)

		var response HealthcheckResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, HealthStatusFail, response.Status)
		assert.Equal(t, map[string]int{
			HealthStatusPass: 2,
			HealthStatusFail: 1,
			HealthStatusWarn: 0,
		}, response.Summary)
		assert.Contains(t, response.Checks, "redis")
		assert.Contains(t, response.Checks, "websocket")
		assert.Contains(t, response.Checks, "metrics")
		assert.Greater(t, response.Timestamp, int64(0))
	})

	// Test unified health endpoint returns success when all components pass
	t.Run("unified health endpoint all pass", func(t *testing.T) {
		allPassCheckers := []HealthChecker{
			&MockHealthChecker{
				name: "redis",
				status: HealthStatus{
					Status:  HealthStatusPass,
					Message: "Redis OK",
				},
			},
			&MockHealthChecker{
				name: "websocket",
				status: HealthStatus{
					Status:  HealthStatusPass,
					Message: "WebSocket OK",
				},
			},
		}

		api := NewHealthcheckAPI(allPassCheckers...)
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		api.Handle().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		body, err := io.ReadAll(rec.Body)
		require.NoError(t, err)

		var response HealthcheckResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, HealthStatusPass, response.Status)
	})
}
