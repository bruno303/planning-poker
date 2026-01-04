package http_test

import (
	"context"
	"net/http"
	"planning-poker/test/integration"
	"testing"
	"time"
)

func TestHealthcheck(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	t.Run("GET /health returns 200", func(t *testing.T) {
		var response struct {
			Status string `json:"status"`
		}

		resp, err := ts.GetJSON(t, "/health", &response)
		if err != nil {
			t.Fatalf("failed to get health: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		if response.Status != "ok" {
			t.Errorf("expected status 'ok', got '%s'", response.Status)
		}
	})

	t.Run("healthcheck responds quickly", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", ts.Server.URL+"/health", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if duration > 50*time.Millisecond {
			t.Errorf("health check took too long: %v (expected < 50ms)", duration)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}
