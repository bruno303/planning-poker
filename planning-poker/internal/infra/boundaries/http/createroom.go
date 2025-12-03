package http

import (
	"encoding/json"
	"net/http"
	"planning-poker/internal/domain"

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
		hub    domain.Hub
		logger log.Logger
	}
)

var _ API = (*CreateRoomAPI)(nil)

func NewCreateRoomAPI(hub domain.Hub) CreateRoomAPI {
	return CreateRoomAPI{
		hub:    hub,
		logger: log.NewLogger("createroomapi"),
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
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		room := c.hub.NewRoom(ctx, body.CreatedBy)
		c.logger.Info(ctx, "New room created: %v", room.ID)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateRoomResponse{RoomID: room.ID})
	})
}
