package entity

import (
	"errors"
	"fmt"
)

type (
	DeckID   string
	DeckKind string

	Deck struct {
		ID    DeckID
		Name  string
		Kind  DeckKind
		Cards []Card
	}

	Card struct {
		Value        string
		NumericValue *int
		Special      bool
	}
)

const (
	DeckIDFibonacci DeckID = "fibonacci"
	DeckIDTShirt    DeckID = "tshirt"
	DefaultDeckID          = DeckIDFibonacci

	DeckKindNumeric     DeckKind = "numeric"
	DeckKindCategorical DeckKind = "categorical"
)

var ErrUnknownDeckID = errors.New("unknown deck ID")

var presetDecks = map[DeckID]Deck{
	DeckIDFibonacci: {
		ID:   DeckIDFibonacci,
		Name: "Fibonacci",
		Kind: DeckKindNumeric,
		Cards: []Card{
			numericCard("0", 0),
			numericCard("1", 1),
			numericCard("2", 2),
			numericCard("3", 3),
			numericCard("5", 5),
			numericCard("8", 8),
			numericCard("13", 13),
			numericCard("21", 21),
			numericCard("34", 34),
			numericCard("55", 55),
			numericCard("89", 89),
			{Value: "?", Special: true},
			{Value: "☕", Special: true},
		},
	},
	DeckIDTShirt: {
		ID:   DeckIDTShirt,
		Name: "T-shirt sizes",
		Kind: DeckKindCategorical,
		Cards: []Card{
			{Value: "XS"},
			{Value: "S"},
			{Value: "M"},
			{Value: "L"},
			{Value: "XL"},
			{Value: "XXL"},
			{Value: "?", Special: true},
			{Value: "☕", Special: true},
		},
	},
}

func numericCard(value string, number int) Card {
	return Card{Value: value, NumericValue: &number}
}

// GetDeck resolves a preset deck, defaulting an empty ID to Fibonacci.
// Returned card slices and numeric metadata are copies of the preset data.
func GetDeck(deckID DeckID) (Deck, error) {
	if deckID == "" {
		deckID = DefaultDeckID
	}
	deck, ok := presetDecks[deckID]
	if !ok {
		return Deck{}, fmt.Errorf("%w: %q", ErrUnknownDeckID, deckID)
	}
	return copyDeck(deck), nil
}

func copyDeck(deck Deck) Deck {
	cards := make([]Card, len(deck.Cards))
	for i, card := range deck.Cards {
		cards[i] = card
		if card.NumericValue != nil {
			numericValue := *card.NumericValue
			cards[i].NumericValue = &numericValue
		}
	}
	deck.Cards = cards
	return deck
}

// Card returns the card matching value in this deck.
func (d Deck) Card(value string) (Card, bool) {
	for _, card := range d.Cards {
		if card.Value == value {
			return card, true
		}
	}
	return Card{}, false
}

// CardValues returns the deck's ordered card values as a new slice.
func (d Deck) CardValues() []string {
	values := make([]string, len(d.Cards))
	for i, card := range d.Cards {
		values[i] = card.Value
	}
	return values
}
