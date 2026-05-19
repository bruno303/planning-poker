package http

import (
	"errors"
	"net/http"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/http/middleware"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/mux"
)

type GetRoomStateAPI struct {
	hub                 domain.Hub
	adminAuthMiddleware middleware.AdminMiddleware
	logger              log.Logger
}

var _ API = (*GetRoomStateAPI)(nil)

// @Summary Get room state
// @Description Returns the state of a specific room (admin only)
// @Tags admin
// @Produce json
// @Param roomID path string true "Room ID"
// @Success 200 {object} GetRoomStateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/rooms/{roomID} [get]
func NewGetRoomStateAPI(hub domain.Hub, adminAuthMiddleware middleware.AdminMiddleware) GetRoomStateAPI {
	return GetRoomStateAPI{
		hub:                 hub,
		adminAuthMiddleware: adminAuthMiddleware,
		logger:              log.NewLogger("getroomstateapi"),
	}
}

func (api GetRoomStateAPI) Endpoint() string {
	return "/admin/rooms/{roomID}"
}

func (api GetRoomStateAPI) Methods() []string {
	return []string{"GET"}
}

func (api GetRoomStateAPI) Handle() http.Handler {
	return api.adminAuthMiddleware.Handle(api.execute())
}

func (api GetRoomStateAPI) execute() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["roomID"]
		if roomID == "" {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		room, err := api.hub.LoadRoom(ctx, roomID)
		if errors.Is(err, domain.ErrRoomNotFound) {
			SendJsonErrorMsg(w, http.StatusNotFound, "Room not found")
			return
		}
		if err != nil {
			api.logger.Error(ctx, "Failed to load room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to load room")
			return
		}

		SendJsonResponse(w, http.StatusOK, mapRoom(room))
	})
}

func mapRoom(room *entity.Room) GetRoomStateResponse {
	return GetRoomStateResponse{
		ID:      room.ID,
		Clients: mapClients(room.Clients),
	}
}
