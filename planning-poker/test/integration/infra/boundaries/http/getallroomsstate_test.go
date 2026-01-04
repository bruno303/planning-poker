package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"planning-poker/test/integration"
	"testing"
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
		// Create first room
		room1Body := map[string]string{"createdBy": "user1"}
		jsonBody1, _ := json.Marshal(room1Body)
		resp1, err := http.Post(
			ts.Server.URL+"/planning/room",
			"application/json",
			bytes.NewBuffer(jsonBody1),
		)
		if err != nil {
			t.Fatalf("failed to create room 1: %v", err)
		}
		defer func() { _ = resp1.Body.Close() }()

		var createResp1 struct {
			RoomID string `json:"roomId"`
		}
		if err := json.NewDecoder(resp1.Body).Decode(&createResp1); err != nil {
			t.Fatalf("failed to decode room 1 response: %v", err)
		}

		// Create second room
		room2Body := map[string]string{"createdBy": "user2"}
		jsonBody2, _ := json.Marshal(room2Body)
		resp2, err := http.Post(
			ts.Server.URL+"/planning/room",
			"application/json",
			bytes.NewBuffer(jsonBody2),
		)
		if err != nil {
			t.Fatalf("failed to create room 2: %v", err)
		}
		defer func() { _ = resp2.Body.Close() }()

		var createResp2 struct {
			RoomID string `json:"roomId"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&createResp2); err != nil {
			t.Fatalf("failed to decode room 2 response: %v", err)
		}

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

		if !roomIDs[createResp1.RoomID] {
			t.Errorf("room 1 ID %s not found in response", createResp1.RoomID)
		}
		if !roomIDs[createResp2.RoomID] {
			t.Errorf("room 2 ID %s not found in response", createResp2.RoomID)
		}
	})

	t.Run("GET /admin/rooms includes client information", func(t *testing.T) {
		// Create a room
		roomBody := map[string]string{"createdBy": "admin-user"}
		jsonBody, _ := json.Marshal(roomBody)
		respCreate, err := http.Post(
			ts.Server.URL+"/planning/room",
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			t.Fatalf("failed to create room: %v", err)
		}
		defer func() { _ = respCreate.Body.Close() }()

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
