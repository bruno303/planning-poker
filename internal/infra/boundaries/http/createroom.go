package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain/domainerror"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomRequest struct {
		DeckPreset string `json:"deckPreset"`
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
// @Description Creates a new planning poker room and returns its ID
// @Tags rooms
// @Accept json
// @Produce json
// @Param request body CreateRoomRequest false "Room options"
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
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Malformed request body")
			return
		}
		output, err := c.createRoom.Execute(r.Context(), usecase.CreateRoomCommand{DeckID: request.DeckPreset})
		if err != nil {
			if errors.Is(err, domainerror.ErrUnknownDeck) {
				SendJsonErrorMsg(w, http.StatusBadRequest, "Unknown deck preset")
				return
			}
			c.logger.Error(r.Context(), "Failed to create room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to create room")
			return
		}

		SendJsonResponse(w, http.StatusCreated, CreateRoomResponse{RoomID: output.RoomID})
	})
}
