package http

import (
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomResponse struct {
		RoomID string `json:"roomId"`
	}
	CreateRoomAPI struct {
		createRoom usecase.UseCaseO[usecase.CreateRoomOutput]
		logger     log.Logger
	}
)

var _ API = (*CreateRoomAPI)(nil)

// @Summary Create a new room
// @Description Creates a new planning poker room and returns its ID
// @Tags rooms
// @Accept json
// @Produce json
// @Success 200 {object} CreateRoomResponse
// @Failure 500 {object} ErrorResponse
// @Router /planning/rooms [post]
func NewCreateRoomAPI(createRoom usecase.UseCaseO[usecase.CreateRoomOutput]) CreateRoomAPI {
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
		output, err := c.createRoom.Execute(r.Context())
		if err != nil {
			c.logger.Error(r.Context(), "Failed to create room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to create room")
			return
		}

		SendJsonResponse(w, http.StatusCreated, CreateRoomResponse{RoomID: output.RoomID})
	})
}
