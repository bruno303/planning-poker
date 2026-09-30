package entity

import (
	"errors"
	"reflect"
	"testing"
)

func TestGetDeckPresets(t *testing.T) {
	tests := []struct {
		name     string
		deckID   DeckID
		wantKind DeckKind
		wantName string
		want     []string
	}{
		{
			name:     "default fibonacci",
			deckID:   "",
			wantKind: DeckKindNumeric,
			wantName: "Fibonacci",
			want:     []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"},
		},
		{
			name:     "t-shirt",
			deckID:   DeckIDTShirt,
			wantKind: DeckKindCategorical,
			wantName: "T-shirt sizes",
			want:     []string{"XS", "S", "M", "L", "XL", "XXL", "?", "☕"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deck, err := GetDeck(tt.deckID)
			if err != nil {
				t.Fatalf("GetDeck returned error: %v", err)
			}
			if deck.Kind != tt.wantKind || deck.Name != tt.wantName || !reflect.DeepEqual(deck.CardValues(), tt.want) {
				t.Fatalf("deck = %+v, want kind %q name %q cards %v", deck, tt.wantKind, tt.wantName, tt.want)
			}
		})
	}
}

func TestGetDeckReturnsCopies(t *testing.T) {
	first, err := GetDeck(DeckIDFibonacci)
	if err != nil {
		t.Fatal(err)
	}
	*first.Cards[0].NumericValue = 99
	first.Cards[0].Value = "mutated"

	second, err := GetDeck(DeckIDFibonacci)
	if err != nil {
		t.Fatal(err)
	}
	if got := second.Cards[0]; got.Value != "0" || got.NumericValue == nil || *got.NumericValue != 0 {
		t.Fatalf("preset was mutated through returned copy: %+v", got)
	}
}

func TestGetDeckRejectsUnknownID(t *testing.T) {
	if _, err := GetDeck("custom"); !errors.Is(err, ErrUnknownDeckID) {
		t.Fatalf("GetDeck error = %v, want ErrUnknownDeckID", err)
	}
}
