package http

import (
	"net/http"

	"planning-poker/internal/domain/entity"
)

type (
	DeckResponse struct {
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Cards []string `json:"cards"`
	}
	GetDecksResponse struct {
		Decks []DeckResponse `json:"decks"`
	}
	GetDecksAPI struct{}
)

var _ API = (*GetDecksAPI)(nil)

// @Summary List available decks
// @Description Returns the backend-owned voting deck catalogue
// @Tags decks
// @Produce json
// @Success 200 {object} GetDecksResponse
// @Router /planning/decks [get]
func NewGetDecksAPI() GetDecksAPI {
	return GetDecksAPI{}
}

func (g GetDecksAPI) Endpoint() string {
	return "/planning/decks"
}

func (g GetDecksAPI) Methods() []string {
	return []string{"GET"}
}

func (g GetDecksAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		SendJsonResponse(w, http.StatusOK, GetDecksResponse{Decks: mapDecks(entity.Decks())})
	})
}

func mapDecks(decks []entity.Deck) []DeckResponse {
	mapped := make([]DeckResponse, len(decks))
	for i, deck := range decks {
		mapped[i] = DeckResponse{
			ID:    deck.ID,
			Name:  deck.Name,
			Cards: deck.Cards,
		}
	}
	return mapped
}
