package dto

import "planning-poker/internal/domain/entity"

type DeckPreset struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Cards []string `json:"cards"`
}

func NewDeckPresets() []DeckPreset {
	decks := entity.DeckPresets()
	result := make([]DeckPreset, len(decks))
	for i, deck := range decks {
		result[i] = DeckPreset{ID: deck.ID, Name: deck.Name, Cards: deck.Cards}
	}
	return result
}
