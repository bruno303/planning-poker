package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"planning-poker/test/integration"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Helper function to create a room and return the room ID
func createRoom(t *testing.T, ts *integration.TestServer) string {
	t.Helper()

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

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var response struct {
		RoomID string `json:"roomId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return response.RoomID
}

// Helper function to connect to WebSocket
func connectWebSocket(t *testing.T, ts *integration.TestServer, roomID string) *websocket.Conn {
	t.Helper()

	wsURL := strings.Replace(ts.Server.URL, "http://", "ws://", 1)
	wsURL = fmt.Sprintf("%s/planning/%s/ws", wsURL, roomID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to websocket: %v", err)
	}

	return conn
}

// Helper to receive a message with timeout
func receiveMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) map[string]any {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(timeout))

	var msg map[string]any
	err := conn.ReadJSON(&msg)
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	return msg
}

// Helper to send a message
func sendMessage(t *testing.T, conn *websocket.Conn, msg map[string]any) {
	t.Helper()

	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
}

// Helper to extract client ID from first message
func getClientID(t *testing.T, conn *websocket.Conn) string {
	t.Helper()

	// First message: update-client-id
	msg1 := receiveMessage(t, conn, 2*time.Second)
	if msg1["type"] != "update-client-id" {
		t.Fatalf("expected first message type 'update-client-id', got '%v'", msg1["type"])
	}
	clientID, ok := msg1["clientId"].(string)
	if !ok || clientID == "" {
		t.Fatal("clientId not found in update-client-id message")
	}

	return clientID
}

func TestWebSocketConnection(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	t.Run("successful connection to existing room", func(t *testing.T) {
		roomID := createRoom(t, ts)
		conn := connectWebSocket(t, ts, roomID)
		defer conn.Close()

		// Should receive update-client-id message
		msg1 := receiveMessage(t, conn, 2*time.Second)
		if msg1["type"] != "update-client-id" {
			t.Errorf("expected first message type 'update-client-id', got '%v'", msg1["type"])
		}
		if msg1["clientId"] == nil {
			t.Error("clientId should not be nil")
		}
	})

	t.Run("connection to non-existent room fails", func(t *testing.T) {
		wsURL := strings.Replace(ts.Server.URL, "http://", "ws://", 1)
		wsURL = fmt.Sprintf("%s/planning/non-existent-room/ws", wsURL)

		dialer := websocket.Dialer{
			HandshakeTimeout: 2 * time.Second,
		}

		conn, resp, err := dialer.Dial(wsURL, nil)
		if err == nil {
			defer conn.Close()
		}

		if err == nil && resp.StatusCode == http.StatusOK {
			t.Error("expected connection to non-existent room to fail")
		}
	})
}

func TestWebSocketUpdateName(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer conn.Close()

	clientID := getClientID(t, conn)

	t.Run("update name successfully", func(t *testing.T) {
		sendMessage(t, conn, map[string]any{
			"type":     "update-name",
			"roomId":   roomID,
			"clientId": clientID,
			"username": "Alice",
		})

		// Should receive room-state update
		msg := receiveMessage(t, conn, 2*time.Second)
		if msg["type"] != "room-state" {
			t.Errorf("expected room-state message, got '%v'", msg["type"])
		}

		// Verify the participant has the updated name
		participants, ok := msg["participants"].([]any)
		if !ok || len(participants) == 0 {
			t.Fatal("expected participants array")
		}

		participant := participants[0].(map[string]any)
		if participant["name"] != "Alice" {
			t.Errorf("expected name 'Alice', got '%v'", participant["name"])
		}
	})
}

func TestWebSocketVoting(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)

	// Connect first client (owner)
	conn1 := connectWebSocket(t, ts, roomID)
	defer conn1.Close()
	clientID1 := getClientID(t, conn1)

	// Connect second client
	conn2 := connectWebSocket(t, ts, roomID)
	defer conn2.Close()
	clientID2 := getClientID(t, conn2)

	t.Run("client can vote", func(t *testing.T) {
		sendMessage(t, conn1, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID1,
			"vote":     "5",
		})

		// Both clients should receive room-state update
		msg1 := receiveMessage(t, conn1, 2*time.Second)
		if msg1["type"] != "room-state" {
			t.Errorf("expected room-state message, got '%v'", msg1["type"])
		}

		msg2 := receiveMessage(t, conn2, 2*time.Second)
		if msg2["type"] != "room-state" {
			t.Errorf("expected room-state message, got '%v'", msg2["type"])
		}

		// Vote should not be revealed yet
		if msg1["reveal"] != false {
			t.Error("votes should not be revealed yet")
		}
	})

	t.Run("votes auto-reveal when all clients vote", func(t *testing.T) {
		sendMessage(t, conn2, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID2,
			"vote":     "8",
		})

		// Both clients should receive room-state update with reveal=true
		msg1 := receiveMessage(t, conn1, 2*time.Second)
		msg2 := receiveMessage(t, conn2, 2*time.Second)

		if msg1["reveal"] != true {
			t.Error("votes should be auto-revealed after all clients vote")
		}
		if msg2["reveal"] != true {
			t.Error("votes should be auto-revealed after all clients vote")
		}

		// Result should be calculated (5+8)/2 = 6.5
		result, ok := msg1["result"].(float64)
		if !ok || result != 6.5 {
			t.Errorf("expected result 6.5, got %v", msg1["result"])
		}
	})
}

func TestWebSocketRevealVotes(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer conn.Close()

	clientID := getClientID(t, conn)

	t.Run("owner can manually reveal votes", func(t *testing.T) {
		// Vote first
		sendMessage(t, conn, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID,
			"vote":     "3",
		})
		// When there's only 1 client, voting auto-reveals, so consume that message
		msg := receiveMessage(t, conn, 2*time.Second)

		// Verify it was auto-revealed
		if msg["reveal"] != true {
			t.Error("votes should be auto-revealed when single client votes")
		}

		// Now toggle reveal off
		sendMessage(t, conn, map[string]any{
			"type":     "reveal-votes",
			"roomId":   roomID,
			"clientId": clientID,
		})

		msg = receiveMessage(t, conn, 2*time.Second)
		if msg["reveal"] != false {
			t.Error("votes should be hidden after toggling reveal")
		}
	})
}

func TestWebSocketReset(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer conn.Close()

	clientID := getClientID(t, conn)

	t.Run("owner can reset voting", func(t *testing.T) {
		// Vote and reveal
		sendMessage(t, conn, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID,
			"vote":     "5",
		})
		_ = receiveMessage(t, conn, 2*time.Second) // consume room-state

		// Reset
		sendMessage(t, conn, map[string]any{
			"type":     "reset",
			"roomId":   roomID,
			"clientId": clientID,
		})

		msg := receiveMessage(t, conn, 2*time.Second)
		if msg["reveal"] != false {
			t.Error("votes should not be revealed after reset")
		}

		participants := msg["participants"].([]any)
		participant := participants[0].(map[string]any)
		if participant["hasVoted"] != false {
			t.Error("participant should not have voted after reset")
		}
	})
}

func TestWebSocketToggleSpectator(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)

	// Connect owner
	conn1 := connectWebSocket(t, ts, roomID)

	defer conn1.Close()
	ownerID := getClientID(t, conn1)

	// Connect second client
	conn2 := connectWebSocket(t, ts, roomID)

	defer conn2.Close()
	clientID2 := getClientID(t, conn2)

	t.Run("owner can toggle spectator mode", func(t *testing.T) {
		sendMessage(t, conn1, map[string]any{
			"type":           "toggle-spectator",
			"roomId":         roomID,
			"clientId":       ownerID,
			"targetClientId": clientID2,
		})

		// Both clients should receive room-state update
		msg1 := receiveMessage(t, conn1, 2*time.Second)

		// Find the target client in participants
		participants := msg1["participants"].([]any)
		var targetParticipant map[string]any
		for _, p := range participants {
			participant := p.(map[string]any)
			if participant["id"] == clientID2 {
				targetParticipant = participant
				break
			}
		}

		if targetParticipant == nil {
			t.Fatal("target participant not found")
		}

		if targetParticipant["isSpectator"] != true {
			t.Error("participant should be marked as spectator")
		}
	})
}

func TestWebSocketToggleOwner(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)

	// Connect owner
	conn1 := connectWebSocket(t, ts, roomID)

	defer conn1.Close()
	ownerID := getClientID(t, conn1)

	// Connect second client
	conn2 := connectWebSocket(t, ts, roomID)

	defer conn2.Close()
	clientID2 := getClientID(t, conn2)

	t.Run("owner can promote another user to owner", func(t *testing.T) {
		sendMessage(t, conn1, map[string]any{
			"type":           "toggle-owner",
			"roomId":         roomID,
			"clientId":       ownerID,
			"targetClientId": clientID2,
		})

		// Both clients should receive room-state update
		msg1 := receiveMessage(t, conn1, 2*time.Second)

		// Find the target client in participants
		participants := msg1["participants"].([]any)
		var targetParticipant map[string]any
		for _, p := range participants {
			participant := p.(map[string]any)
			if participant["id"] == clientID2 {
				targetParticipant = participant
				break
			}
		}

		if targetParticipant == nil {
			t.Fatal("target participant not found")
		}

		if targetParticipant["isOwner"] != true {
			t.Error("participant should be promoted to owner")
		}
	})
}

func TestWebSocketUpdateStory(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer conn.Close()

	clientID := getClientID(t, conn)

	t.Run("owner can update current story", func(t *testing.T) {
		sendMessage(t, conn, map[string]any{
			"type":     "update-story",
			"roomId":   roomID,
			"clientId": clientID,
			"story":    "User story #123",
		})

		msg := receiveMessage(t, conn, 2*time.Second)
		if msg["currentStory"] != "User story #123" {
			t.Errorf("expected currentStory 'User story #123', got '%v'", msg["currentStory"])
		}
	})
}

func TestWebSocketNewVoting(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer conn.Close()

	clientID := getClientID(t, conn)

	t.Run("owner can start new voting", func(t *testing.T) {
		// Vote first
		sendMessage(t, conn, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID,
			"vote":     "5",
		})
		_ = receiveMessage(t, conn, 2*time.Second) // consume room-state

		// Set story
		sendMessage(t, conn, map[string]any{
			"type":     "update-story",
			"roomId":   roomID,
			"clientId": clientID,
			"story":    "Old story",
		})
		_ = receiveMessage(t, conn, 2*time.Second) // consume room-state

		// Start new voting
		sendMessage(t, conn, map[string]any{
			"type":     "new-voting",
			"roomId":   roomID,
			"clientId": clientID,
		})

		msg := receiveMessage(t, conn, 2*time.Second)

		// Story should be cleared
		if msg["currentStory"] != "" {
			t.Errorf("expected currentStory to be empty, got '%v'", msg["currentStory"])
		}

		// Votes should be cleared
		if msg["reveal"] != false {
			t.Error("votes should not be revealed after new voting")
		}

		participants := msg["participants"].([]any)
		participant := participants[0].(map[string]any)
		if participant["hasVoted"] != false {
			t.Error("participant should not have voted after new voting")
		}
	})
}

func TestWebSocketVoteAgain(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer conn.Close()

	clientID := getClientID(t, conn)

	t.Run("owner can trigger vote again", func(t *testing.T) {
		// Vote first
		sendMessage(t, conn, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID,
			"vote":     "5",
		})
		_ = receiveMessage(t, conn, 2*time.Second) // consume room-state

		// Vote again
		sendMessage(t, conn, map[string]any{
			"type":     "vote-again",
			"roomId":   roomID,
			"clientId": clientID,
		})

		msg := receiveMessage(t, conn, 2*time.Second)

		// Votes should be cleared but story preserved
		if msg["reveal"] != false {
			t.Error("votes should not be revealed after vote again")
		}

		participants := msg["participants"].([]any)
		participant := participants[0].(map[string]any)
		if participant["hasVoted"] != false {
			t.Error("participant should not have voted after vote again")
		}
	})
}

func TestWebSocketMultipleClients(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)

	// Connect 3 clients
	conn1 := connectWebSocket(t, ts, roomID)
	defer conn1.Close()
	clientID1 := getClientID(t, conn1)

	conn2 := connectWebSocket(t, ts, roomID)
	defer conn2.Close()
	clientID2 := getClientID(t, conn2)

	conn3 := connectWebSocket(t, ts, roomID)
	defer conn3.Close()
	clientID3 := getClientID(t, conn3)

	t.Run("all clients receive updates when one client votes", func(t *testing.T) {
		sendMessage(t, conn1, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID1,
			"vote":     "3",
		})

		// All three clients should receive the update
		msg1 := receiveMessage(t, conn1, 2*time.Second)
		msg2 := receiveMessage(t, conn2, 2*time.Second)
		msg3 := receiveMessage(t, conn3, 2*time.Second)

		for _, msg := range []map[string]any{msg1, msg2, msg3} {
			if msg["type"] != "room-state" {
				t.Error("all clients should receive room-state update")
			}

			participants := msg["participants"].([]any)
			if len(participants) != 3 {
				t.Errorf("expected 3 participants, got %d", len(participants))
			}
		}
	})

	t.Run("votes auto-reveal when all 3 clients vote", func(t *testing.T) {
		// Reset voting first to clear the vote from client1 in the previous test
		sendMessage(t, conn1, map[string]any{
			"type":     "reset",
			"roomId":   roomID,
			"clientId": clientID1,
		})
		// Consume reset messages
		_ = receiveMessage(t, conn1, 2*time.Second)
		_ = receiveMessage(t, conn2, 2*time.Second)
		_ = receiveMessage(t, conn3, 2*time.Second)

		// Now have all 3 clients vote
		sendMessage(t, conn1, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID1,
			"vote":     "3",
		})
		// Consume vote messages
		_ = receiveMessage(t, conn1, 2*time.Second)
		_ = receiveMessage(t, conn2, 2*time.Second)
		_ = receiveMessage(t, conn3, 2*time.Second)

		// Client 2 votes
		sendMessage(t, conn2, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID2,
			"vote":     "5",
		})
		// Consume room-state updates from all clients
		_ = receiveMessage(t, conn1, 2*time.Second)
		_ = receiveMessage(t, conn2, 2*time.Second)
		_ = receiveMessage(t, conn3, 2*time.Second)

		// Client 3 votes - should trigger auto-reveal
		sendMessage(t, conn3, map[string]any{
			"type":     "vote",
			"roomId":   roomID,
			"clientId": clientID3,
			"vote":     "8",
		})

		// Now get the reveal messages
		msg1 := receiveMessage(t, conn1, 2*time.Second)
		msg2 := receiveMessage(t, conn2, 2*time.Second)
		msg3 := receiveMessage(t, conn3, 2*time.Second)

		for _, msg := range []map[string]any{msg1, msg2, msg3} {
			if msg["reveal"] != true {
				t.Error("votes should be auto-revealed when all clients vote")
			}

			// Result should be (3+5+8)/3 = 5.333...
			result, ok := msg["result"].(float64)
			if !ok {
				t.Error("result should be present")
			}
			expectedResult := (3.0 + 5.0 + 8.0) / 3.0
			// Use a small tolerance for floating point comparison
			tolerance := 0.001
			if result < expectedResult-tolerance || result > expectedResult+tolerance {
				t.Errorf("expected result %.2f, got %.2f", expectedResult, result)
			}
		}
	})
}

func TestWebSocketDisconnection(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)

	t.Run("client can disconnect gracefully", func(t *testing.T) {
		// Connect two clients first so room doesn't get deleted when one leaves
		conn1 := connectWebSocket(t, ts, roomID)
		_ = getClientID(t, conn1)

		conn2 := connectWebSocket(t, ts, roomID)
		clientID2 := getClientID(t, conn2)

		// Close the first connection
		err := conn1.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			t.Logf("error sending close message: %v", err)
		}
		conn1.Close()

		// Give server time to process disconnection
		time.Sleep(100 * time.Millisecond)

		// Verify the second client is still connected and the room still exists
		// We can verify this by sending a message
		sendMessage(t, conn2, map[string]any{
			"type":     "update-name",
			"roomId":   roomID,
			"clientId": clientID2,
			"username": "Still Connected",
		})

		// Should receive room-state update
		msg := receiveMessage(t, conn2, 2*time.Second)
		if msg["type"] != "room-state" {
			t.Errorf("expected room-state message, got '%v'", msg["type"])
		}

		conn2.Close()
	})
}
