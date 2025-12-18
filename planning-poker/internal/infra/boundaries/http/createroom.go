package http

import (
	"encoding/json"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomRequest struct {
		CreatedBy string `json:"createdBy"`
	}
	CreateRoomResponse struct {
		RoomID string `json:"roomId"`
	}
	CreateRoomAPI struct {
		usecase usecase.UseCaseWithResult[usecase.CreateRoomCommand, usecase.CreateRoomOutput]
		logger  log.Logger
	}
)

var _ API = (*CreateRoomAPI)(nil)

func NewCreateRoomAPI(usecase usecase.UseCaseWithResult[usecase.CreateRoomCommand, usecase.CreateRoomOutput]) CreateRoomAPI {
	return CreateRoomAPI{
		usecase: usecase,
		logger:  log.NewLogger("createroomapi"),
	}
}

func (c CreateRoomAPI) Endpoint() string {
	return "/planning/room"
}

func (c CreateRoomAPI) Methods() []string {
	return []string{"POST", "OPTIONS"}
}

func (c CreateRoomAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var body CreateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		output, err := c.usecase.Execute(ctx, usecase.CreateRoomCommand{SenderID: body.CreatedBy})
		if err != nil {
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to create room")
			return
		}

		c.logger.Info(ctx, "New room created: %v", output.Room.ID)
		SendJsonResponse(w, http.StatusCreated, CreateRoomResponse{RoomID: output.Room.ID})
	})
}
