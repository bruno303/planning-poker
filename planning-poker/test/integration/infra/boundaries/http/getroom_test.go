package http_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"planning-poker/test/integration"
	"testing"
	"time"
)

// TestGetRoom tests retrieving room state
func TestGetRoom(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	// First create a room via WebSocket auto-creation and keep the
	// connection alive so the room isn't cleaned up by the hub.
	roomID := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
	conn := connectWebSocket(t, ts, roomID)
	defer closeAndWait(conn)
	_ = getClientID(t, conn)

	t.Run("GET /planning/room/{id} returns room state", func(t *testing.T) {
		resp, err := http.Get(ts.Server.URL + "/planning/room/" + roomID)
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
		if roomState.RoomID != roomID {
			t.Errorf("expected roomId %s, got %s", roomID, roomState.RoomID)
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
