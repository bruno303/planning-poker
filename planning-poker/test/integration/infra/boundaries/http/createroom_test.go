package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"planning-poker/test/integration"
	"testing"
)

func TestCreateRoom(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	t.Run("POST /planning/room creates a new room", func(t *testing.T) {
		requestBody := map[string]string{
			"createdBy": "test-user",
		}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			ts.Server.URL+"/planning/room",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusCreated)

		var response struct {
			RoomID string `json:"roomId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.RoomID == "" {
			t.Error("expected roomId to be non-empty")
		}
	})

	t.Run("POST /planning/room with empty body still succeeds", func(t *testing.T) {
		requestBody := map[string]string{}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			ts.Server.URL+"/planning/room",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusCreated)

		var response struct {
			RoomID string `json:"roomId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.RoomID == "" {
			t.Error("expected roomId to be non-empty")
		}
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		resp, err := http.Post(
			ts.Server.URL+"/planning/room",
			"application/json",
			bytes.NewBuffer([]byte("not valid json")),
		)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusBadRequest)
	})
}
