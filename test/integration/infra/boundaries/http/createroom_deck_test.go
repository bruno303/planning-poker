package http_test

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"planning-poker/internal/domain/entity"
	"planning-poker/test/integration"
)

func TestCreateRoomWithDeckPreset(t *testing.T) {
	ts := integration.NewTestServer(t)
	defer ts.Close()

	client := integration.NewHTTPClient(ts.Server.URL)

	t.Run("POST /planning/rooms with a tshirt preset persists the deck", func(t *testing.T) {
		var response struct {
			RoomID string `json:"roomId"`
		}
		resp, err := client.PostJSON(t, "/planning/rooms", map[string]string{"deckPreset": "tshirt"}, &response)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		integration.AssertStatus(t, resp, http.StatusCreated)

		if response.RoomID == "" {
			t.Fatal("expected a room ID in the response")
		}

		room, err := ts.Container.Infra.Hub.LoadRoom(context.Background(), response.RoomID)
		if err != nil {
			t.Fatalf("failed to load created room: %v", err)
		}

		tshirtDeck, ok := entity.DeckByID("tshirt")
		if !ok {
			t.Fatal("tshirt deck preset not found")
		}
		if room.Deck.ID != "tshirt" {
			t.Fatalf("expected room deck tshirt, got %q", room.Deck.ID)
		}
		if !slices.Equal(room.Deck.Cards, tshirtDeck.Cards) {
			t.Fatalf("room deck cards = %v, want %v", room.Deck.Cards, tshirtDeck.Cards)
		}
	})

	t.Run("POST /planning/rooms with an unknown preset returns 400", func(t *testing.T) {
		resp, err := client.Post(t, "/planning/rooms", map[string]string{"deckPreset": "unknown"})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		integration.AssertStatus(t, resp, http.StatusBadRequest)
	})
}
