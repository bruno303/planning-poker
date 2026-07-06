package http_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"planning-poker/test/integration"
	"testing"
	"time"
)

func TestGetAllRoomsState(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	// Get the API key from test config (config-test.yaml)
	apiKey := "my-secret-key"

	t.Run("GET /admin/rooms without auth returns 401", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.Server.URL+"/admin/rooms", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("GET /admin/rooms with invalid API key returns 401", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.Server.URL+"/admin/rooms", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-key")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("GET /admin/rooms with valid API key returns empty list when no rooms", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.Server.URL+"/admin/rooms", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusOK)

		var rooms []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(rooms) != 0 {
			t.Errorf("expected 0 rooms, got %d", len(rooms))
		}
	})

	t.Run("GET /admin/rooms returns list of rooms after creating them", func(t *testing.T) {
		// Create first room via WebSocket and keep connection alive so the
		// room isn't cleaned up by the hub.
		roomID1 := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
		conn1 := connectWebSocket(t, ts, roomID1)
		defer closeAndWait(conn1)
		_ = getClientID(t, conn1)

		// Create second room
		roomID2 := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
		conn2 := connectWebSocket(t, ts, roomID2)
		defer closeAndWait(conn2)
		_ = getClientID(t, conn2)

		// Now get all rooms
		req, err := http.NewRequest("GET", ts.Server.URL+"/admin/rooms", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusOK)

		var rooms []struct {
			ID      string `json:"ID"`
			Clients []struct {
				ID          string `json:"ID"`
				Name        string `json:"Name"`
				IsSpectator bool   `json:"IsSpectator"`
				IsOwner     bool   `json:"IsOwner"`
			} `json:"Clients"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(rooms) != 2 {
			t.Errorf("expected 2 rooms, got %d", len(rooms))
		}

		// Verify room IDs are in the response
		roomIDs := make(map[string]bool)
		for _, room := range rooms {
			roomIDs[room.ID] = true
		}

		if !roomIDs[roomID1] {
			t.Errorf("room 1 ID %s not found in response", roomID1)
		}
		if !roomIDs[roomID2] {
			t.Errorf("room 2 ID %s not found in response", roomID2)
		}
	})

	t.Run("GET /admin/rooms includes client information", func(t *testing.T) {
		// Create a room via WebSocket and keep connection alive so the room
		// persists for the admin query.
		roomID := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
		conn := connectWebSocket(t, ts, roomID)
		defer closeAndWait(conn)
		_ = getClientID(t, conn)

		// Get all rooms
		req, err := http.NewRequest("GET", ts.Server.URL+"/admin/rooms", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusOK)

		var rooms []struct {
			ID      string `json:"ID"`
			Clients []struct {
				ID          string `json:"ID"`
				Name        string `json:"Name"`
				IsSpectator bool   `json:"IsSpectator"`
				IsOwner     bool   `json:"IsOwner"`
			} `json:"Clients"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// At least one room should exist
		if len(rooms) == 0 {
			t.Fatal("expected at least one room")
		}

		// Each room should have clients array (even if empty)
		for i, room := range rooms {
			if room.Clients == nil {
				t.Errorf("room %d clients array is nil", i)
			}
		}
	})

	t.Run("GET /admin/rooms with Authorization header without Bearer prefix returns 401", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.Server.URL+"/admin/rooms", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "InvalidPrefix "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusUnauthorized)
	})
}
