package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"planning-poker/internal/domain/entity"
)

func TestGetDecksAPI_Endpoint(t *testing.T) {
	api := NewGetDecksAPI()
	if api.Endpoint() != "/planning/decks" {
		t.Errorf("Endpoint() = %v, want /planning/decks", api.Endpoint())
	}
}

func TestGetDecksAPI_Methods(t *testing.T) {
	api := NewGetDecksAPI()
	methods := api.Methods()
	if len(methods) != 1 || methods[0] != http.MethodGet {
		t.Errorf("Methods() = %v, want [GET]", methods)
	}
}

func TestGetDecksAPI_Handle_ReturnsCatalogue(t *testing.T) {
	api := NewGetDecksAPI()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/planning/decks", nil)

	api.Handle().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response GetDecksResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response.Decks) != 2 {
		t.Fatalf("deck count = %d, want 2", len(response.Decks))
	}
	if response.Decks[0].ID != string(entity.DeckTypeFibonacci) || response.Decks[1].ID != string(entity.DeckTypeTShirt) {
		t.Errorf("deck IDs = [%s, %s], want [%s, %s]",
			response.Decks[0].ID, response.Decks[1].ID, entity.DeckTypeFibonacci, entity.DeckTypeTShirt)
	}
	if response.Decks[1].Cards[0] != "XS" {
		t.Errorf("t-shirt first card = %q, want XS", response.Decks[1].Cards[0])
	}
}
