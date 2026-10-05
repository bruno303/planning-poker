package entity

import (
	"reflect"
	"testing"
)

func TestDecksReturnsCatalogueInOrder(t *testing.T) {
	decks := Decks()
	if len(decks) != 2 {
		t.Fatalf("Decks() length = %d, want 2", len(decks))
	}
	if decks[0].ID != string(DeckTypeFibonacci) || decks[1].ID != string(DeckTypeTShirt) {
		t.Fatalf("Decks() order = [%s, %s], want [%s, %s]", decks[0].ID, decks[1].ID, DeckTypeFibonacci, DeckTypeTShirt)
	}
}

func TestDeckDefinitions(t *testing.T) {
	tests := []struct {
		deckType DeckType
		name     string
		kind     DeckKind
		cards    []string
	}{
		{
			deckType: DeckTypeFibonacci,
			name:     "Fibonacci",
			kind:     DeckKindNumeric,
			cards:    []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"},
		},
		{
			deckType: DeckTypeTShirt,
			name:     "T-shirt",
			kind:     DeckKindCategorical,
			cards:    []string{"XS", "S", "M", "L", "XL", "XXL", "?", "☕"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.deckType), func(t *testing.T) {
			deck, ok := DeckByType(tt.deckType)
			if !ok {
				t.Fatalf("DeckByType(%q) not found", tt.deckType)
			}
			if deck.Name != tt.name {
				t.Errorf("deck name = %q, want %q", deck.Name, tt.name)
			}
			if deck.Kind != tt.kind {
				t.Errorf("deck kind = %q, want %q", deck.Kind, tt.kind)
			}
			if !reflect.DeepEqual(deck.Cards, tt.cards) {
				t.Errorf("deck cards = %v, want %v", deck.Cards, tt.cards)
			}
		})
	}
}

func TestDeckByTypeRejectsUnknown(t *testing.T) {
	if _, ok := DeckByType(DeckType("planning")); ok {
		t.Fatal("DeckByType accepted an unknown deck type")
	}
}

func TestDeckHasCardIncludesSpecialCards(t *testing.T) {
	deck, _ := DeckByType(DeckTypeTShirt)

	for _, card := range []string{"XS", "XXL", SpecialVoteUnknown, SpecialVoteCoffee} {
		if !deck.HasCard(card) {
			t.Errorf("HasCard(%q) = false, want true", card)
		}
	}
	if deck.HasCard("5") {
		t.Error("t-shirt deck accepted Fibonacci card 5")
	}
}

func TestDeckPositionExcludesSpecialCards(t *testing.T) {
	deck, _ := DeckByType(DeckTypeTShirt)

	position, ok := deck.Position("XS")
	if !ok || position != 0 {
		t.Errorf("Position(XS) = (%d, %v), want (0, true)", position, ok)
	}
	position, ok = deck.Position("XXL")
	if !ok || position != 5 {
		t.Errorf("Position(XXL) = (%d, %v), want (5, true)", position, ok)
	}
	if _, ok := deck.Position(SpecialVoteCoffee); ok {
		t.Error("Position accepted a special card")
	}
}

func TestDeckEstimateCards(t *testing.T) {
	deck, _ := DeckByType(DeckTypeTShirt)
	if got := deck.EstimateCards(); !reflect.DeepEqual(got, []string{"XS", "S", "M", "L", "XL", "XXL"}) {
		t.Errorf("EstimateCards() = %v", got)
	}
}

func TestDeckForTypeFallsBackToFibonacci(t *testing.T) {
	if got := deckForType(DeckType("")).ID; got != string(DeckTypeFibonacci) {
		t.Errorf("deckForType(empty) = %q, want fibonacci", got)
	}
	if got := DefaultDeck().ID; got != string(DeckTypeFibonacci) {
		t.Errorf("DefaultDeck() = %q, want fibonacci", got)
	}
}

func TestDecksReturnsIsolatedCopies(t *testing.T) {
	decks := Decks()
	decks[0].Cards[0] = "mutated"

	fresh := Decks()
	if fresh[0].Cards[0] == "mutated" {
		t.Fatal("mutating Decks() result changed the backend deck definition")
	}
}
