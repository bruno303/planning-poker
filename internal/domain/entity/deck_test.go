package entity

import (
	"reflect"
	"testing"
)

var (
	fibonacciCards = []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"}
	tshirtCards    = []string{"XS", "S", "M", "L", "XL", "XXL", "?", "☕"}
)

func TestDeckPresets_StableOrder(t *testing.T) {
	decks := DeckPresets()

	if len(decks) != 2 {
		t.Fatalf("DeckPresets() returned %d decks, want 2", len(decks))
	}
	if decks[0].ID != "fibonacci" {
		t.Errorf("DeckPresets()[0].ID = %q, want fibonacci", decks[0].ID)
	}
	if decks[1].ID != "tshirt" {
		t.Errorf("DeckPresets()[1].ID = %q, want tshirt", decks[1].ID)
	}
}

func TestDeckPresets_Values(t *testing.T) {
	decks := DeckPresets()

	tests := []struct {
		index int
		id    string
		name  string
		cards []string
	}{
		{index: 0, id: "fibonacci", name: "Fibonacci", cards: fibonacciCards},
		{index: 1, id: "tshirt", name: "T-shirt sizes", cards: tshirtCards},
	}

	for _, test := range tests {
		deck := decks[test.index]
		if deck.ID != test.id {
			t.Errorf("deck %d ID = %q, want %q", test.index, deck.ID, test.id)
		}
		if deck.Name != test.name {
			t.Errorf("deck %d Name = %q, want %q", test.index, deck.Name, test.name)
		}
		if !reflect.DeepEqual(deck.Cards, test.cards) {
			t.Errorf("deck %d Cards = %v, want %v", test.index, deck.Cards, test.cards)
		}
	}
}

func TestDeckPresets_ReturnsDefensiveCopies(t *testing.T) {
	decks := DeckPresets()
	decks[0].ID = "mutated"
	decks[0].Cards[0] = "mutated"

	again := DeckPresets()
	if again[0].ID != "fibonacci" {
		t.Errorf("DeckPresets() ID was mutated through a returned copy: %q", again[0].ID)
	}
	if again[0].Cards[0] != "0" {
		t.Errorf("DeckPresets() cards were mutated through a returned copy: %q", again[0].Cards[0])
	}
}

func TestDefaultDeck(t *testing.T) {
	deck := DefaultDeck()

	if deck.ID != "fibonacci" {
		t.Errorf("DefaultDeck().ID = %q, want fibonacci", deck.ID)
	}
	if deck.Name != "Fibonacci" {
		t.Errorf("DefaultDeck().Name = %q, want Fibonacci", deck.Name)
	}
	if !reflect.DeepEqual(deck.Cards, fibonacciCards) {
		t.Errorf("DefaultDeck().Cards = %v, want %v", deck.Cards, fibonacciCards)
	}

	deck.Cards[0] = "mutated"
	if DefaultDeck().Cards[0] != "0" {
		t.Error("DefaultDeck() cards were mutated through a returned copy")
	}
}

func TestDeckByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantID  string
		wantOK  bool
		wantLen int
	}{
		{name: "fibonacci", id: "fibonacci", wantID: "fibonacci", wantOK: true, wantLen: len(fibonacciCards)},
		{name: "tshirt", id: "tshirt", wantID: "tshirt", wantOK: true, wantLen: len(tshirtCards)},
		{name: "unknown", id: "unknown", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deck, ok := DeckByID(test.id)
			if ok != test.wantOK {
				t.Fatalf("DeckByID(%q) ok = %v, want %v", test.id, ok, test.wantOK)
			}
			if !test.wantOK {
				if !reflect.DeepEqual(deck, Deck{}) {
					t.Errorf("DeckByID(%q) = %+v, want zero deck", test.id, deck)
				}
				return
			}
			if deck.ID != test.wantID {
				t.Errorf("DeckByID(%q).ID = %q, want %q", test.id, deck.ID, test.wantID)
			}
			if len(deck.Cards) != test.wantLen {
				t.Errorf("DeckByID(%q) has %d cards, want %d", test.id, len(deck.Cards), test.wantLen)
			}
		})
	}
}

func TestDeckByID_ReturnsDefensiveCopy(t *testing.T) {
	deck, ok := DeckByID("tshirt")
	if !ok {
		t.Fatal("tshirt deck preset not found")
	}
	deck.Cards[0] = "mutated"

	again, ok := DeckByID("tshirt")
	if !ok {
		t.Fatal("tshirt deck preset not found")
	}
	if again.Cards[0] != "XS" {
		t.Errorf("DeckByID() cards were mutated through a returned copy: %q", again.Cards[0])
	}
}

func TestDeckContains(t *testing.T) {
	fibonacci, _ := DeckByID("fibonacci")
	tshirt, _ := DeckByID("tshirt")

	tests := []struct {
		name string
		deck Deck
		card string
		want bool
	}{
		{name: "numeric fibonacci card", deck: fibonacci, card: "13", want: true},
		{name: "question mark is on every deck", deck: fibonacci, card: "?", want: true},
		{name: "coffee is on every deck", deck: tshirt, card: "☕", want: true},
		{name: "tshirt card", deck: tshirt, card: "XL", want: true},
		{name: "off-deck numeric card", deck: fibonacci, card: "4", want: false},
		{name: "off-deck tshirt card", deck: tshirt, card: "4", want: false},
		{name: "empty card is not contained", deck: fibonacci, card: "", want: false},
		{name: "empty deck contains nothing", deck: Deck{}, card: "0", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.deck.Contains(test.card); got != test.want {
				t.Errorf("Contains(%q) = %v, want %v", test.card, got, test.want)
			}
		})
	}
}

func TestDeckPosition(t *testing.T) {
	fibonacci, _ := DeckByID("fibonacci")

	tests := []struct {
		name     string
		card     string
		want     int
		wantOK   bool
	}{
		{name: "first card", card: "0", want: 0, wantOK: true},
		{name: "middle card", card: "13", want: 6, wantOK: true},
		{name: "question mark", card: "?", want: 11, wantOK: true},
		{name: "coffee", card: "☕", want: 12, wantOK: true},
		{name: "off-deck card", card: "4", want: 0, wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := fibonacci.Position(test.card)
			if ok != test.wantOK || got != test.want {
				t.Errorf("Position(%q) = (%d, %v), want (%d, %v)", test.card, got, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestDeckClone(t *testing.T) {
	original, _ := DeckByID("tshirt")
	clone := original.Clone()

	if !reflect.DeepEqual(clone, original) {
		t.Fatalf("Clone() = %+v, want %+v", clone, original)
	}

	clone.Cards[0] = "mutated"
	if original.Cards[0] != "XS" {
		t.Errorf("Clone() shares its cards slice with the original: %q", original.Cards[0])
	}
}
