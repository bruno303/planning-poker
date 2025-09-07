package planningpoker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/mux"
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

func ConfigurePlanningPokerAPI(mux *mux.Router, hub *Hub) {
	mux.HandleFunc("/planning/{roomID}/ws", handleConnections(hub))
	mux.HandleFunc("/planning/room", createRoom(hub)).Methods("POST", "OPTIONS")
	mux.HandleFunc("/planning/room/{roomID}", getRoom(hub)).Methods("GET")
}

type contextKey string

func handleConnections(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := mux.Vars(r)["roomID"]
		if roomID == "" {
			http.Error(w, "Room ID is required", http.StatusBadRequest)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), contextKey("roomID"), roomID))
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(r.Context(), "Error upgrading to WebSocket", err)
		}

		room, err := hub.GetRoom(roomID)
		if err != nil {
			logger.Error(r.Context(), "Error getting room", err)
			return
		}

		client := NewClient(ws, room)
		room.AddClient(client)
		logger.Info(r.Context(), "New client connected: %v on room: %v", client.ID, room.ID)

		go client.Listen(r.Context())
		client.Send(r.Context(), NewRoomStateCommand(room))
	}
}

type CreateRoomRequest struct {
	CreatedBy string `json:"createdBy"`
}

type CreateRoomResponse struct {
	RoomID string `json:"roomId"`
}

func createRoom(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var body CreateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		room := hub.NewRoom(body.CreatedBy)
		logger.Info(r.Context(), "New room created: %v", room.ID)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateRoomResponse{RoomID: room.ID})
	}
}

func getRoom(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := mux.Vars(r)["roomID"]
		if roomID == "" {
			http.Error(w, "Room ID is required", http.StatusBadRequest)
			return
		}

		_, err := hub.GetRoom(roomID)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
