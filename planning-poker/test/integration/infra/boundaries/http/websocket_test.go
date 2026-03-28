package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"planning-poker/internal/infra/bus"
	"planning-poker/test/integration"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketConnection(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	t.Run("successful connection to existing room", func(t *testing.T) {
		roomID := createRoom(t, ts)
		conn := connectWebSocket(t, ts, roomID)
		defer closeAndWait(conn)

		// Should receive update-client-id message
		msg1 := receiveMessage(t, conn, 2*time.Second)
		if msg1["type"] != "update-client-id" {
			t.Errorf("expected first message type 'update-client-id', got '%v'", msg1["type"])
		}
		if msg1["clientId"] == nil {
			t.Error("clientId should not be nil")
		}
	})

	t.Run("connection to non-existent room auto-creates room", func(t *testing.T) {
		roomID := "auto-created-room"

		conn1 := connectWebSocket(t, ts, roomID)
		defer closeAndWait(conn1)

		msg1 := receiveMessage(t, conn1, 2*time.Second)
		if msg1["type"] != "update-client-id" {
			t.Fatalf("expected first message type 'update-client-id', got '%v'", msg1["type"])
		}
		clientID1, ok := msg1["clientId"].(string)
		if !ok || clientID1 == "" {
			t.Fatal("clientId not found in update-client-id message")
		}

		msg2 := receiveMessage(t, conn1, 2*time.Second)
		if msg2["type"] != "room-state" {
			t.Fatalf("expected second message type 'room-state', got '%v'", msg2["type"])
		}
		participants1 := msg2["participants"].([]any)
		if len(participants1) != 1 {
			t.Fatalf("expected 1 participant in auto-created room, got %d", len(participants1))
		}
		firstParticipant := participants1[0].(map[string]any)
		if firstParticipant["id"] != clientID1 {
			t.Fatalf("expected first participant id '%s', got '%v'", clientID1, firstParticipant["id"])
		}
		if firstParticipant["isOwner"] != true {
			t.Fatal("expected first auto-created room participant to be owner")
		}

		conn2 := connectWebSocket(t, ts, roomID)
		defer closeAndWait(conn2)

		msg3 := receiveMessage(t, conn2, 2*time.Second)
		if msg3["type"] != "update-client-id" {
			t.Fatalf("expected first message type 'update-client-id', got '%v'", msg3["type"])
		}
		clientID2, ok := msg3["clientId"].(string)
		if !ok || clientID2 == "" {
			t.Fatal("clientId not found in update-client-id message")
		}

		msg4 := receiveMessage(t, conn2, 2*time.Second)
		if msg4["type"] != "room-state" {
			t.Fatalf("expected second message type 'room-state', got '%v'", msg4["type"])
		}

		msg5 := receiveMessage(t, conn1, 2*time.Second)
		if msg5["type"] != "room-state" {
			t.Fatalf("expected broadcast message type 'room-state', got '%v'", msg5["type"])
		}

		for _, msg := range []map[string]any{msg4, msg5} {
			participants := msg["participants"].([]any)
			if len(participants) != 2 {
				t.Fatalf("expected 2 participants after second join, got %d", len(participants))
			}

			seenClient1 := false
			seenClient2 := false
			for _, p := range participants {
				participant := p.(map[string]any)
				switch participant["id"] {
				case clientID1:
					seenClient1 = true
				case clientID2:
					seenClient2 = true
				}
			}

			if !seenClient1 || !seenClient2 {
				t.Fatalf("expected both auto-created room participants to be present, got %#v", participants)
			}
		}
	})
}

func TestWebSocketUpdateName(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer closeAndWait(conn)

	_ = getClientID(t, conn)

	t.Run("update name successfully", func(t *testing.T) {
		send(t, conn, bus.WebSocketMessage{
			Type: "update-name",
			Payload: bus.UpdateNamePayload{
				Username: "Alice",
			},
		})

		// Should receive room-state update
		msgs := readMessages(t, conn)
		msg := msgs[0]
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
	defer closeAndWait(conn1)
	_ = getClientID(t, conn1)

	// Connect second client
	conn2 := connectWebSocket(t, ts, roomID)
	defer closeAndWait(conn2)
	_ = getClientID(t, conn2)
	// Client1 receives broadcast when client2 joins
	consumeMessages(t, conn1)

	t.Run("client can vote", func(t *testing.T) {
		send(t, conn1, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "5",
			},
		})

		// Both clients should receive room-state update
		msgs := readMessages(t, conn1, conn2)

		for _, msg := range msgs {
			if msg["type"] != "room-state" {
				t.Error("both clients should receive room-state update")
			}
		}

		// Vote should not be revealed yet
		if msgs[0]["reveal"] != false {
			t.Error("votes should not be revealed yet")
		}
	})

	t.Run("votes auto-reveal when all clients vote", func(t *testing.T) {
		send(t, conn2, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "8",
			},
		})

		// Both clients should receive room-state update with reveal=true
		msgs := readMessages(t, conn1, conn2)
		for _, msg := range msgs {
			if msg["reveal"] != true {
				t.Error("votes should be auto-revealed after all clients vote")
			}
		}

		// Result should be calculated (5+8)/2 = 6.5
		result, ok := msgs[0]["result"].(float64)
		if !ok || result != 6.5 {
			t.Errorf("expected result 6.5, got %v", msgs[0]["result"])
		}
	})
}

func TestWebSocketRevealVotes(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	roomID := createRoom(t, ts)
	conn := connectWebSocket(t, ts, roomID)

	defer closeAndWait(conn)

	_ = getClientID(t, conn)

	t.Run("owner can manually reveal votes", func(t *testing.T) {
		// Vote first
		send(t, conn, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "3",
			},
		})
		// When there's only 1 client, voting auto-reveals, so consume that message
		msg := readMessages(t, conn)[0]

		// Verify it was auto-revealed
		if msg["reveal"] != true {
			t.Error("votes should be auto-revealed when single client votes")
		}

		// Now toggle reveal off
		send(t, conn, bus.WebSocketMessage{
			Type: "reveal-votes",
		})

		msg = readMessages(t, conn)[0]
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

	defer closeAndWait(conn)

	_ = getClientID(t, conn)

	t.Run("owner can reset voting", func(t *testing.T) {
		// Vote and reveal
		send(t, conn, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "5",
			},
		})
		consumeMessages(t, conn) // consume room-state

		// Reset
		send(t, conn, bus.WebSocketMessage{
			Type: "reset",
		})

		msg := readMessages(t, conn)[0]
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

	defer closeAndWait(conn1)
	_ = getClientID(t, conn1)

	// Connect second client
	conn2 := connectWebSocket(t, ts, roomID)

	defer closeAndWait(conn2)
	clientID2 := getClientID(t, conn2)
	// Client1 receives broadcast when client2 joins
	consumeMessages(t, conn1)

	t.Run("owner can toggle spectator mode", func(t *testing.T) {
		send(t, conn1, bus.WebSocketMessage{
			Type: "toggle-spectator",
			Payload: bus.ToggleSpectatorPayload{
				TargetClientID: clientID2,
			},
		})

		// Both clients should receive room-state update
		msg1 := readMessages(t, conn1)[0]

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

	defer closeAndWait(conn1)
	_ = getClientID(t, conn1)

	// Connect second client
	conn2 := connectWebSocket(t, ts, roomID)

	defer closeAndWait(conn2)
	clientID2 := getClientID(t, conn2)
	// Client1 receives broadcast when client2 joins
	consumeMessages(t, conn1)

	t.Run("owner can promote another user to owner", func(t *testing.T) {
		send(t, conn1, bus.WebSocketMessage{
			Type: "toggle-owner",
			Payload: bus.ToggleOwnerPayload{
				TargetClientID: clientID2,
			},
		})

		// Both clients should receive room-state update
		msg1 := readMessages(t, conn1)[0]

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

	defer closeAndWait(conn)

	_ = getClientID(t, conn)

	t.Run("owner can update current story", func(t *testing.T) {
		send(t, conn, bus.WebSocketMessage{
			Type: "update-story",
			Payload: bus.UpdateStoryPayload{
				Story: "User story #123",
			},
		})

		msg := readMessages(t, conn)[0]
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

	defer closeAndWait(conn)

	_ = getClientID(t, conn)

	t.Run("owner can start new voting", func(t *testing.T) {
		// Vote first
		send(t, conn, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "5",
			},
		})
		consumeMessages(t, conn) // consume room-state

		// Set story
		send(t, conn, bus.WebSocketMessage{
			Type: "update-story",
			Payload: bus.UpdateStoryPayload{
				Story: "Old story",
			},
		})
		consumeMessages(t, conn) // consume room-state

		// Start new voting
		send(t, conn, bus.WebSocketMessage{
			Type: "new-voting",
		})

		msg := readMessages(t, conn)[0]

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

	defer closeAndWait(conn)

	_ = getClientID(t, conn)

	t.Run("owner can trigger vote again", func(t *testing.T) {
		// Vote first
		send(t, conn, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "5",
			},
		})
		consumeMessages(t, conn) // consume room-state

		// Vote again
		send(t, conn, bus.WebSocketMessage{
			Type: "vote-again",
		})

		msg := readMessages(t, conn)[0]

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
	defer closeAndWait(conn1)
	_ = getClientID(t, conn1)

	conn2 := connectWebSocket(t, ts, roomID)
	defer closeAndWait(conn2)
	_ = getClientID(t, conn2)
	// Client1 receives broadcast when client2 joins
	consumeMessages(t, conn1)

	conn3 := connectWebSocket(t, ts, roomID)
	defer closeAndWait(conn3)
	_ = getClientID(t, conn3)
	// Client1 and Client2 receive broadcasts when client3 joins
	consumeMessages(t, conn1, conn2)

	t.Run("all clients receive updates when one client votes", func(t *testing.T) {
		send(t, conn1, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "3",
			},
		})

		// All three clients should receive the update
		msgs := readMessages(t, conn1, conn2, conn3)

		for _, msg := range msgs {
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
		send(t, conn1, bus.WebSocketMessage{
			Type: "reset",
		})
		// Consume reset messages
		consumeMessages(t, conn1, conn2, conn3)

		// Now have all 3 clients vote
		send(t, conn1, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "3",
			},
		})
		// Consume vote messages
		consumeMessages(t, conn1, conn2, conn3)

		// Client 2 votes
		send(t, conn2, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "5",
			},
		})
		// Consume room-state updates from all clients
		consumeMessages(t, conn1, conn2, conn3)

		// Client 3 votes - should trigger auto-reveal
		send(t, conn3, bus.WebSocketMessage{
			Type: "vote",
			Payload: bus.VotePayload{
				Vote: "8",
			},
		})

		// Now get the reveal messages
		msgs := readMessages(t, conn1, conn2, conn3)
		for _, msg := range msgs {
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
		_ = getClientID(t, conn2)

		conn1.Close()

		time.Sleep(100 * time.Millisecond)

		// Verify the second client is still connected and the room still exists
		// We can verify this by sending a message
		send(t, conn2, bus.WebSocketMessage{
			Type: "update-name",
			Payload: bus.UpdateNamePayload{
				Username: "Still Connected",
			},
		})

		// Should receive room-state update
		msg := receiveMessage(t, conn2, 2*time.Second)
		if msg["type"] != "room-state" {
			t.Errorf("expected room-state message, got '%v'", msg["type"])
		}

		conn2.Close()
	})

	t.Run("last client disconnect cleans up room for future reconnect", func(t *testing.T) {
		roomID := "disconnect-recreate-room"

		conn1 := connectWebSocket(t, ts, roomID)
		_ = getClientID(t, conn1)
		closeAndWait(conn1)

		conn2 := connectWebSocket(t, ts, roomID)
		defer closeAndWait(conn2)

		msg1 := receiveMessage(t, conn2, 2*time.Second)
		if msg1["type"] != "update-client-id" {
			t.Fatalf("expected first message type 'update-client-id', got '%v'", msg1["type"])
		}

		msg2 := receiveMessage(t, conn2, 2*time.Second)
		if msg2["type"] != "room-state" {
			t.Fatalf("expected second message type 'room-state', got '%v'", msg2["type"])
		}

		participants, ok := msg2["participants"].([]any)
		if !ok {
			t.Fatal("expected participants array")
		}
		if len(participants) != 1 {
			t.Fatalf("expected recreated room to have 1 participant, got %d", len(participants))
		}

		participant := participants[0].(map[string]any)
		if participant["isOwner"] != true {
			t.Fatal("expected reconnected participant to own recreated room")
		}
	})
}

func send(t *testing.T, conn *websocket.Conn, msg bus.WebSocketMessage) {
	t.Helper()

	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
}

func consumeMessages(t *testing.T, conns ...*websocket.Conn) {
	t.Helper()
	readMessages(t, conns...)
}

func readMessages(t *testing.T, conns ...*websocket.Conn) []map[string]any {
	t.Helper()
	messages := make([]map[string]any, len(conns))
	for i, conn := range conns {
		messages[i] = receiveMessage(t, conn, 2*time.Second)
	}
	return messages
}

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

func sendMessage(t *testing.T, conn *websocket.Conn, msg map[string]any) {
	t.Helper()

	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
}

func closeAndWait(conn *websocket.Conn) {
	_ = conn.Close()
	time.Sleep(100 * time.Millisecond)
}

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

	// Consume the initial room-state message that comes after joining
	msg2 := receiveMessage(t, conn, 2*time.Second)
	if msg2["type"] != "room-state" {
		t.Fatalf("expected second message type 'room-state', got '%v'", msg2["type"])
	}

	return clientID
}
