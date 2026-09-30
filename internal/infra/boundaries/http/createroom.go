package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/bruno303/go-toolkit/pkg/log"

	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain/entity"
)

type (
	CreateRoomRequest struct {
		DeckID string `json:"deckId,omitempty" enums:"fibonacci,tshirt"`
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
// @Param request body CreateRoomRequest false "Room creation options"
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
		command, err := parseCreateRoomCommand(r.Body)
		if err != nil {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Invalid room creation request")
			return
		}

		output, err := c.createRoom.Execute(r.Context(), command)
		if err != nil {
			if errors.Is(err, entity.ErrUnknownDeckID) {
				SendJsonErrorMsg(w, http.StatusBadRequest, "Invalid deck ID")
				return
			}
			c.logger.Error(r.Context(), "Failed to create room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to create room")
			return
		}

		SendJsonResponse(w, http.StatusCreated, CreateRoomResponse{RoomID: output.RoomID})
	})
}

func parseCreateRoomCommand(body io.Reader) (usecase.CreateRoomCommand, error) {
	command := usecase.CreateRoomCommand{}
	if body == nil {
		return command, nil
	}

	decoder := json.NewDecoder(body)
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		if errors.Is(err, io.EOF) {
			return command, nil
		}
		return command, err
	}
	if fields == nil {
		return command, errors.New("request body must be an object")
	}
	if rawDeckID, ok := fields["deckId"]; ok {
		if strings.TrimSpace(string(rawDeckID)) == "null" {
			return command, errors.New("deckId must be a string")
		}
		var deckID string
		if err := json.Unmarshal(rawDeckID, &deckID); err != nil {
			return command, err
		}
		command.DeckID = entity.DeckID(deckID)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return command, errors.New("request body contains trailing JSON")
		}
		return command, err
	}
	return command, nil
}
