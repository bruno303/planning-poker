package planningpoker

import (
	"net/http"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var logger = log.NewLogger("planningpoker.api")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ConfigurePlanningPokerAPI(mux *http.ServeMux, hub *Hub) {
	mux.HandleFunc("/planning/ws", handleConnections(hub))
}

func handleConnections(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(r.Context(), "Error upgrading to WebSocket", err)
		}

		room := hub.GetRoom("default")
		defer func() {
			room.RemoveClient("default")
			if len(room.Clients) == 0 {
				hub.RemoveRoom("default")
				logger.Info(r.Context(), "Room 'default' removed from hub")
			}
		}()

		client := NewClient(uuid.NewString(), ws, room)
		room.AddClient(client)
		defer client.Close()

		logger.Info(r.Context(), "New client connected: %v", client.ID)
		client.Listen(r.Context())
	}
}
