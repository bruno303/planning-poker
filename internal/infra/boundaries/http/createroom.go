package http

import (
	"encoding/json"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomRequest struct {
		DeckType string `json:"deckType"`
	}
	CreateRoomResponse struct {
		RoomID string `json:"roomId"`
	}
	CreateRoomAPI struct {
		createRoom usecase.UseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput]
		logger     log.Logger
	}
)

var _ API = (*CreateRoomAPI)(nil)

// @Summary Create a new room
// @Description Creates a new planning poker room with the selected deck and returns its ID
// @Tags rooms
// @Accept json
// @Produce json
// @Param request body CreateRoomRequest true "Room creation request"
// @Success 201 {object} CreateRoomResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /planning/rooms [post]
func NewCreateRoomAPI(createRoom usecase.UseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput]) CreateRoomAPI {
	return CreateRoomAPI{
		createRoom: createRoom,
		logger:     log.NewLogger("createroomapi"),
	}
}

func (c CreateRoomAPI) Endpoint() string {
	return "/planning/rooms"
}

func (c CreateRoomAPI) Methods() []string {
	return []string{"POST"}
}

func (c CreateRoomAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		deckType := entity.DeckType(request.DeckType)
		if request.DeckType == "" {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Deck type is required")
			return
		}
		if _, ok := entity.DeckByType(deckType); !ok {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Unknown deck type")
			return
		}

		output, err := c.createRoom.Execute(r.Context(), usecase.CreateRoomCommand{DeckType: deckType})
		if err != nil {
			c.logger.Error(r.Context(), "Failed to create room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to create room")
			return
		}

		SendJsonResponse(w, http.StatusCreated, CreateRoomResponse{RoomID: output.RoomID})
	})
}
