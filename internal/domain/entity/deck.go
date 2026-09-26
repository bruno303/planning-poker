package entity

// Deck is an ordered set of cards available for voting in a room.
type Deck struct {
	ID    string
	Name  string
	Cards []string
}

var deckPresets = []Deck{
	{ID: "fibonacci", Name: "Fibonacci", Cards: []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"}},
	{ID: "tshirt", Name: "T-shirt sizes", Cards: []string{"XS", "S", "M", "L", "XL", "XXL", "?", "☕"}},
}

func (d Deck) Contains(card string) bool {
	for _, candidate := range d.Cards {
		if candidate == card {
			return true
		}
	}
	return false
}

func (d Deck) Position(card string) (int, bool) {
	for i, candidate := range d.Cards {
		if candidate == card {
			return i, true
		}
	}
	return 0, false
}

func (d Deck) Clone() Deck {
	d.Cards = append([]string(nil), d.Cards...)
	return d
}

func DefaultDeck() Deck { return deckPresets[0].Clone() }

func DeckByID(id string) (Deck, bool) {
	for _, deck := range deckPresets {
		if deck.ID == id {
			return deck.Clone(), true
		}
	}
	return Deck{}, false
}

func DeckPresets() []Deck {
	decks := make([]Deck, len(deckPresets))
	for i, deck := range deckPresets {
		decks[i] = deck.Clone()
	}
	return decks
}
