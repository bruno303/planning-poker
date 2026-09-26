package http

import (
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase/dto"
)

type ListDecksAPI struct{}

var _ API = (*ListDecksAPI)(nil)

func NewListDecksAPI() ListDecksAPI    { return ListDecksAPI{} }
func (ListDecksAPI) Endpoint() string  { return "/planning/decks" }
func (ListDecksAPI) Methods() []string { return []string{"GET"} }
func (ListDecksAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		SendJsonResponse(w, http.StatusOK, map[string]any{"decks": dto.NewDeckPresets()})
	})
}
