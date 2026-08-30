package http_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"planning-poker/internal/infra/boundaries/hub/inmemory"
	"planning-poker/internal/infra/bus"
	"planning-poker/test/integration"

	"github.com/gorilla/websocket"
)

func TestInMemoryWebSocketConcurrentJoins(t *testing.T) {
	ts := integration.NewInMemoryTestServer(t)
	defer ts.Close()

	const clientCount = 8
	roomID := "concurrent-join-room"
	start := make(chan struct{})
	connections := make(chan connectionResult, clientCount)
	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			conn := connectWebSocket(t, ts, roomID)
			clientID := clientIDFromUpdateMessage(t, receiveMessage(t, conn, 2*time.Second))
			connections <- connectionResult{conn: conn, clientID: clientID}
		}()
	}
	close(start)
	wg.Wait()
	close(connections)

	clients := make([]connectionResult, 0, clientCount)
	defer func() {
		for _, client := range clients {
			_ = client.conn.Close()
		}
	}()
	for client := range connections {
		clients = append(clients, client)
	}

	ids := make([]string, 0, clientCount)
	for _, client := range clients {
		ids = append(ids, client.clientID)
	}
	for _, client := range clients {
		var msg map[string]any
		for i := 0; i < clientCount; i++ {
			msg = receiveMessage(t, client.conn, 2*time.Second)
			if len(msg["participants"].([]any)) == clientCount {
				break
			}
		}
		assertRoomStateMessage(t, msg, clientCount)
		assertParticipants(t, msg["participants"], ids...)
	}
}

type connectionResult struct {
	conn     *websocket.Conn
	clientID string
}

func TestInMemoryWebSocketConcurrentGuardedCommands(t *testing.T) {
	ts := integration.NewInMemoryTestServer(t)
	defer ts.Close()

	roomID := "concurrent-command-room"
	owner := connectWebSocket(t, ts, roomID)
	defer owner.Close()
	_ = getClientID(t, owner)
	other := connectWebSocket(t, ts, roomID)
	defer other.Close()
	otherID := getClientID(t, other)
	consumeMessages(t, owner)
	send(t, owner, bus.WebSocketMessage{
		Type: "toggle-owner",
		Payload: bus.ToggleOwnerPayload{
			TargetClientID:      otherID,
			ExpectedRoomVersion: uint64Ptr(0),
		},
	})
	consumeMessages(t, owner, other)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, conn := range []*websocket.Conn{owner, other} {
		wg.Add(1)
		go func(conn *websocket.Conn) {
			defer wg.Done()
			<-start
			send(t, conn, bus.WebSocketMessage{
				Type: "update-story",
				Payload: bus.UpdateStoryPayload{
					Story:               "concurrent story",
					ExpectedRoomVersion: uint64Ptr(1),
				},
			})
		}(conn)
	}
	close(start)
	wg.Wait()

	assertGuardedCommandResults(t, owner, other)
}

func assertGuardedCommandResults(t *testing.T, connections ...*websocket.Conn) {
	t.Helper()
	messages := make(chan map[string]any, 3)
	for _, conn := range connections {
		go readUntilStaleCommand(conn, messages)
	}

	roomStates := 0
	staleCommands := 0
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for roomStates < 2 || staleCommands < 1 {
		select {
		case message := <-messages:
			switch message["type"] {
			case "room-state":
				roomStates++
				assertCommittedRoomState(t, message)
			case "stale-command":
				staleCommands++
				if message["roomVersion"] != float64(2) {
					t.Errorf("expected stale command to report room version 2, got %v", message["roomVersion"])
				}
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for guarded command results: room states=%d stale commands=%d", roomStates, staleCommands)
		}
	}
}

func assertCommittedRoomState(t *testing.T, message map[string]any) {
	t.Helper()
	if message["currentStory"] != "concurrent story" {
		t.Errorf("expected committed story %q, got %v", "concurrent story", message["currentStory"])
	}
	if message["roomVersion"] != float64(2) {
		t.Errorf("expected committed room version 2, got %v", message["roomVersion"])
	}
}

func readUntilStaleCommand(conn *websocket.Conn, messages chan<- map[string]any) {
	for {
		var message map[string]any
		if err := conn.ReadJSON(&message); err != nil {
			return
		}
		messages <- message
		if message["type"] == "stale-command" {
			return
		}
	}
}

func TestInMemoryWebSocketBroadcastAndCleanup(t *testing.T) {
	ts := integration.NewInMemoryTestServer(t)
	defer ts.Close()

	roomID := fmt.Sprintf("cleanup-room-%d", time.Now().UnixNano())
	first := connectWebSocket(t, ts, roomID)
	firstID := getClientID(t, first)
	second := connectWebSocket(t, ts, roomID)
	defer second.Close()
	_ = getClientID(t, second)
	consumeMessages(t, first)

	if err := first.Close(); err != nil {
		t.Fatalf("failed to close first client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	hub, ok := ts.Container.Infra.Hub.(*inmemory.InMemoryHub)
	if !ok {
		t.Fatal("expected in-memory hub")
	}
	if err := hub.WaitForClientRemoval(ctx, roomID, firstID); err != nil {
		t.Fatalf("waiting for first client cleanup: %v", err)
	}
	send(t, second, bus.WebSocketMessage{Type: "update-name", Payload: bus.UpdateNamePayload{Username: "retained"}})
	if message := receiveMessage(t, second, 2*time.Second); message["type"] != "room-state" {
		t.Fatalf("expected retained client to receive broadcast, got %v", message["type"])
	}

	if err := second.Close(); err != nil {
		t.Fatalf("failed to close second client: %v", err)
	}
	if err := hub.WaitForRoomRemoval(ctx, roomID); err != nil {
		t.Fatalf("waiting for room cleanup: %v", err)
	}
	recreated := connectWebSocket(t, ts, roomID)
	defer recreated.Close()
	_ = clientIDFromUpdateMessage(t, receiveMessage(t, recreated, 2*time.Second))
	message := receiveMessage(t, recreated, 2*time.Second)
	assertRoomStateMessage(t, message, 1)
}
