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
			http.Error(w, "Room ID is required", http.StatusBadRequest)
			return
		}

		_, ok := g.hub.GetRoom(ctx, roomID)
		if !ok {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
