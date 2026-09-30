package http_test

import (
	"net/http"
	"testing"
	"time"

	"planning-poker/internal/infra/bus"
	"planning-poker/test/integration"
)

func TestHTTPDeckCreationAndWebSocketLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		deckID     string
		deckName   string
		cards      []string
		firstVote  string
		secondVote string
		result     float64
		modes      []string
	}{
		{
			name:       "default Fibonacci deck",
			deckID:     "fibonacci",
			deckName:   "Fibonacci",
			cards:      []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"},
			firstVote:  "5",
			secondVote: "8",
			result:     6.5,
			modes:      []string{"5", "8"},
		},
		{
			name:       "selected T-shirt deck",
			deckID:     "tshirt",
			deckName:   "T-shirt sizes",
			cards:      []string{"XS", "S", "M", "L", "XL", "XXL", "?", "☕"},
			firstVote:  "M",
			secondVote: "L",
			modes:      []string{"M", "L"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := integration.NewTestServer(t)
			defer ts.Close()

			body := map[string]string{}
			if tt.deckID != "fibonacci" {
				body["deckId"] = tt.deckID
			}
			var created struct {
				RoomID string `json:"roomId"`
			}
			response, err := integration.NewHTTPClient(ts.Server.URL).PostJSON(t, "/planning/rooms", body, &created)
			if err != nil {
				t.Fatalf("create room: %v", err)
			}
			integration.AssertStatus(t, response, http.StatusCreated)
			if created.RoomID == "" {
				t.Fatal("room creation returned an empty room ID")
			}

			owner := connectWebSocket(t, ts, created.RoomID)
			defer closeAndWait(owner)
			clientIDFromUpdateMessage(t, receiveMessage(t, owner, 2*time.Second))
			assertDeckState(t, receiveMessage(t, owner, 2*time.Second), tt.deckID, tt.deckName, tt.cards)

			guest := connectWebSocket(t, ts, created.RoomID)
			defer closeAndWait(guest)
			clientIDFromUpdateMessage(t, receiveMessage(t, guest, 2*time.Second))
			assertDeckState(t, receiveMessage(t, guest, 2*time.Second), tt.deckID, tt.deckName, tt.cards)
			assertDeckState(t, receiveMessage(t, owner, 2*time.Second), tt.deckID, tt.deckName, tt.cards)

			send(t, owner, bus.WebSocketMessage{Type: "vote", Payload: bus.VotePayload{Vote: tt.firstVote}})
			for _, state := range readMessages(t, owner, guest) {
				if state["reveal"] != false {
					t.Fatalf("votes revealed before both clients voted: %#v", state)
				}
				assertDeckState(t, state, tt.deckID, tt.deckName, tt.cards)
			}

			send(t, guest, bus.WebSocketMessage{Type: "vote", Payload: bus.VotePayload{Vote: tt.secondVote}})
			for _, state := range readMessages(t, owner, guest) {
				if state["reveal"] != true {
					t.Fatalf("votes not revealed after both clients voted: %#v", state)
				}
				assertDeckState(t, state, tt.deckID, tt.deckName, tt.cards)
				assertStringValues(t, state["mostCommonVotes"], tt.modes)
				if tt.deckID == "fibonacci" {
					if got, ok := state["result"].(float64); !ok || got != tt.result {
						t.Errorf("Fibonacci result = %v, want %v", state["result"], tt.result)
					}
				} else if _, exists := state["result"]; exists {
					t.Errorf("T-shirt room must not have a numeric result, got %v", state["result"])
				}
			}
		})
	}
}

func assertDeckState(t *testing.T, state map[string]any, expectedID string, expectedName string, expectedCards []string) {
	t.Helper()
	if state["type"] != "room-state" {
		t.Fatalf("expected room-state message, got %v", state["type"])
	}
	deck, ok := state["deck"].(map[string]any)
	if !ok {
		t.Fatalf("room state has no deck descriptor: %#v", state["deck"])
	}
	if deck["id"] != expectedID || deck["name"] != expectedName {
		t.Errorf("deck descriptor = (%v, %v), want (%s, %s)", deck["id"], deck["name"], expectedID, expectedName)
	}
	assertStringValues(t, deck["cards"], expectedCards)
}

func assertStringValues(t *testing.T, value any, expected []string) {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("expected string list %v, got %#v", expected, value)
	}
	actual := make([]string, 0, len(values))
	for _, value := range values {
		item, ok := value.(string)
		if !ok {
			t.Fatalf("expected string list %v, got %#v", expected, value)
		}
		actual = append(actual, item)
	}
	if len(actual) != len(expected) {
		t.Errorf("values = %v, want %v", actual, expected)
		return
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Errorf("values = %v, want %v", actual, expected)
			return
		}
	}
}
