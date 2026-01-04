package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"planning-poker/test/integration"
	"testing"
)

// TestGetRoom tests retrieving room state
func TestGetRoom(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	// First create a room
	requestBody := map[string]string{
		"createdBy": "test-user",
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	createResp, err := http.Post(
		ts.Server.URL+"/planning/room",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("create room failed: %v", err)
	}
	defer func() { _ = createResp.Body.Close() }()

	var createResponse struct {
		RoomID string `json:"roomId"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createResponse); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	t.Run("GET /planning/room/{id} returns room state", func(t *testing.T) {
		resp, err := http.Get(ts.Server.URL + "/planning/room/" + createResponse.RoomID)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusOK)

		var roomState struct {
			RoomID string `json:"roomId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&roomState); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Verify room ID is returned
		if roomState.RoomID != createResponse.RoomID {
			t.Errorf("expected roomId %s, got %s", createResponse.RoomID, roomState.RoomID)
		}
	})

	t.Run("GET non-existent room returns 404", func(t *testing.T) {
		resp, err := http.Get(ts.Server.URL + "/planning/room/non-existent-id")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusNotFound)
	})
}
