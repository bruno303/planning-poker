package entity

// Deck identifies a supported voting deck.
type Deck string

const (
	DeckFibonacci Deck = "fibonacci"
	DeckTShirt    Deck = "t-shirt"
)

var decks = map[Deck][]string{
	DeckFibonacci: {"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"},
	DeckTShirt:    {"XS", "S", "M", "L", "XL", "XXL", "?", "☕"},
}

// Valid reports whether the deck is supported.
func (d Deck) Valid() bool { _, ok := decks[d]; return ok }

// Labels returns a copy of the deck's ordered labels.
func (d Deck) Labels() []string { return append([]string(nil), decks[d]...) }
