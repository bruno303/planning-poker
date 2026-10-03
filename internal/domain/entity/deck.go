package entity

import "slices"

// DeckType identifies a backend-owned voting deck.
type DeckType string

const (
	DeckTypeFibonacci DeckType = "fibonacci"
	DeckTypeTShirt    DeckType = "t-shirt"
)

const (
	// SpecialVoteUnknown is the "?" special card.
	SpecialVoteUnknown = "?"
	// SpecialVoteCoffee is the "☕" special card.
	SpecialVoteCoffee = "☕"
)

// Deck is an immutable ordered set of voting cards. Cards lists the ordered
// estimate labels followed by the shared special cards.
type Deck struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Cards []string `json:"cards"`
}

var specialDeckCards = []string{SpecialVoteUnknown, SpecialVoteCoffee}

var (
	fibonacciDeck = Deck{
		ID:    string(DeckTypeFibonacci),
		Name:  "Fibonacci",
		Cards: append([]string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89"}, specialDeckCards...),
	}
	tShirtDeck = Deck{
		ID:    string(DeckTypeTShirt),
		Name:  "T-shirt",
		Cards: append([]string{"XS", "S", "M", "L", "XL", "XXL"}, specialDeckCards...),
	}
	decks = []Deck{fibonacciDeck, tShirtDeck}
)

// Decks returns the backend-owned deck catalogue in presentation order.
func Decks() []Deck {
	cloned := make([]Deck, len(decks))
	for i, deck := range decks {
		cloned[i] = deck.clone()
	}
	return cloned
}

// DeckByType resolves a deck descriptor by its type.
func DeckByType(deckType DeckType) (Deck, bool) {
	for _, deck := range decks {
		if deck.ID == string(deckType) {
			return deck.clone(), true
		}
	}
	return Deck{}, false
}

// DefaultDeck returns the Fibonacci deck used by zero-value rooms and the
// join-by-link auto-creation path.
func DefaultDeck() Deck {
	return fibonacciDeck.clone()
}

func deckForType(deckType DeckType) Deck {
	if deck, ok := DeckByType(deckType); ok {
		return deck
	}
	return DefaultDeck()
}

func (d Deck) clone() Deck {
	cloned := d
	cloned.Cards = append([]string(nil), d.Cards...)
	return cloned
}

// HasCard reports whether the vote is one of the deck's cards, including
// special cards.
func (d Deck) HasCard(vote string) bool {
	return slices.Contains(d.Cards, vote)
}

// IsSpecial reports whether the vote is a special card excluded from ordered
// summaries.
func (d Deck) IsSpecial(vote string) bool {
	return slices.Contains(specialDeckCards, vote)
}

// EstimateCards returns the ordered estimate cards without special cards.
func (d Deck) EstimateCards() []string {
	estimates := make([]string, 0, len(d.Cards))
	for _, card := range d.Cards {
		if d.IsSpecial(card) {
			break
		}
		estimates = append(estimates, card)
	}
	return estimates
}

// Position returns the index of vote among the ordered estimate cards.
func (d Deck) Position(vote string) (int, bool) {
	for index, card := range d.EstimateCards() {
		if card == vote {
			return index, true
		}
	}
	return 0, false
}
