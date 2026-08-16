package http_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"planning-poker/test/integration"
	"testing"
	"time"
)

const adminAPIKey = "my-secret-key"

func newAdminRoomsRequest(t *testing.T, ts *integration.TestServer, authorization string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, ts.Server.URL+"/admin/rooms", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return req
}

func TestGetAllRoomsState(t *testing.T) {
	t.Run("without auth returns 401", testGetAllRoomsWithoutAuth)
	t.Run("with invalid API key returns 401", testGetAllRoomsInvalidAPIKey)
	t.Run("with valid API key returns empty list", testGetAllRoomsEmpty)
	t.Run("returns rooms after creating them", testGetAllRoomsAfterCreating)
	t.Run("includes client information", testGetAllRoomsIncludesClients)
	t.Run("requires Bearer prefix", testGetAllRoomsRequiresBearer)
}

func testGetAllRoomsWithoutAuth(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	resp, err := http.DefaultClient.Do(newAdminRoomsRequest(t, ts, ""))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	integration.AssertStatus(t, resp, http.StatusUnauthorized)
}

func testGetAllRoomsInvalidAPIKey(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	resp, err := http.DefaultClient.Do(newAdminRoomsRequest(t, ts, "Bearer invalid-key"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	integration.AssertStatus(t, resp, http.StatusUnauthorized)
}

func testGetAllRoomsEmpty(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	resp, err := http.DefaultClient.Do(newAdminRoomsRequest(t, ts, "Bearer "+adminAPIKey))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	integration.AssertStatus(t, resp, http.StatusOK)

	var rooms []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(rooms) != 0 {
		t.Errorf("expected 0 rooms, got %d", len(rooms))
	}
}

func testGetAllRoomsAfterCreating(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID1 := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
	conn1 := connectWebSocket(t, ts, roomID1)
	defer closeAndWait(conn1)
	_ = getClientID(t, conn1)

	roomID2 := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
	conn2 := connectWebSocket(t, ts, roomID2)
	defer closeAndWait(conn2)
	_ = getClientID(t, conn2)

	resp, err := http.DefaultClient.Do(newAdminRoomsRequest(t, ts, "Bearer "+adminAPIKey))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	integration.AssertStatus(t, resp, http.StatusOK)

	var rooms []struct {
		ID string `json:"ID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rooms); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}

	roomIDs := make(map[string]bool)
	for _, room := range rooms {
		roomIDs[room.ID] = true
	}
	if !roomIDs[roomID1] || !roomIDs[roomID2] {
		t.Errorf("room IDs not found in response: %v", roomIDs)
	}
}

func testGetAllRoomsIncludesClients(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
	conn := connectWebSocket(t, ts, roomID)
	defer closeAndWait(conn)
	_ = getClientID(t, conn)

	resp, err := http.DefaultClient.Do(newAdminRoomsRequest(t, ts, "Bearer "+adminAPIKey))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	integration.AssertStatus(t, resp, http.StatusOK)

	var rooms []struct {
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
	if len(rooms) == 0 {
		t.Fatal("expected at least one room")
	}
	for i, room := range rooms {
		if room.Clients == nil {
			t.Errorf("room %d clients array is nil", i)
		}
	}
}

func testGetAllRoomsRequiresBearer(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	resp, err := http.DefaultClient.Do(newAdminRoomsRequest(t, ts, "InvalidPrefix "+adminAPIKey))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	integration.AssertStatus(t, resp, http.StatusUnauthorized)
}
