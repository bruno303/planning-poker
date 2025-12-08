package http

import (
	"net/http"
	"planning-poker/internal/domain"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/mux"
)

type (
	GetRoomResponse struct {
		RoomID string `json:"roomId"`
	}
	GetRoomAPI struct {
		hub    domain.Hub
		logger log.Logger
	}
)

var _ API = (*GetRoomAPI)(nil)

func NewGetRoomAPI(hub domain.Hub) GetRoomAPI {
	return GetRoomAPI{
		hub:    hub,
		logger: log.NewLogger("getroomapi"),
	}
}

func (g GetRoomAPI) Endpoint() string {
	return "/planning/room/{roomID}"
}

func (g GetRoomAPI) Methods() []string {
	return []string{"GET"}
}

func (g GetRoomAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["roomID"]
		if roomID == "" {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		_, ok := g.hub.GetRoom(ctx, roomID)
		if !ok {
			SendJsonErrorMsg(w, http.StatusNotFound, "Room not found")
			return
		}

		SendJsonResponse(w, http.StatusOK, GetRoomResponse{RoomID: roomID})
	})
}
