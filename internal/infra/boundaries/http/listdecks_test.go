package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListDecksAPI_EndpointAndMethods(t *testing.T) {
	api := NewListDecksAPI()

	if got := api.Endpoint(); got != "/planning/decks" {
		t.Fatalf("Endpoint() = %q, want %q", got, "/planning/decks")
	}
	if got := api.Methods(); len(got) != 1 || got[0] != http.MethodGet {
		t.Fatalf("Methods() = %v, want [GET]", got)
	}
}

func TestListDecksAPI_Handle_ReturnsPresetsInStableOrder(t *testing.T) {
	api := NewListDecksAPI()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/planning/decks", nil)

	api.Handle().ServeHTTP(recorder, request)

	want := `{"decks":[` +
		`{"id":"fibonacci","name":"Fibonacci","cards":["0","1","2","3","5","8","13","21","34","55","89","?","☕"]},` +
		`{"id":"tshirt","name":"T-shirt sizes","cards":["XS","S","M","L","XL","XXL","?","☕"]}` +
		`]}`
	assertJSONResponse(t, recorder, http.StatusOK, want)
}
