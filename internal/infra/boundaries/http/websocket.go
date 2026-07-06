package http

import (
	"context"
	"fmt"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/infra/bus"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type (
	contextKey string

	WebsocketAPI struct {
		upgrader   websocket.Upgrader
		usecases   usecase.UseCasesFacade
		busFactory *bus.WebSocketBusFactory
		logger     log.Logger
	}
)

var _ API = (*WebsocketAPI)(nil)

// @Summary WebSocket connection
// @Description Upgrades the HTTP connection to a WebSocket for real-time communication
// @Tags rooms
// @Param roomID path string true "Room ID"
// @Success 101 {string} string "WebSocket upgrade successful"
// @Router /planning/{roomID}/ws [get]
func NewWebsocketAPI(usecases usecase.UseCasesFacade, websocketBusFactory *bus.WebSocketBusFactory) *WebsocketAPI {
	return &WebsocketAPI{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		usecases:   usecases,
		busFactory: websocketBusFactory,
		logger:     log.NewLogger("planningpoker.api.websocket"),
	}
}

func (api *WebsocketAPI) Endpoint() string {
	return "/planning/{roomID}/ws"
}

func (api *WebsocketAPI) Methods() []string {
	return nil
}

func (api *WebsocketAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roomID := mux.Vars(r)["roomID"]
		if roomID == "" {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Room ID is required")
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), contextKey("roomID"), roomID))
		ws, err := api.upgrader.Upgrade(w, r, nil)
		if err != nil {
			api.logger.Error(r.Context(), "Error upgrading to WebSocket", err)
			return
		}

		clientID := r.URL.Query().Get("clientId")
		if clientID == "" {
			createClientOutput, err := api.usecases.CreateClient.Execute(r.Context())
			if err != nil {
				api.logger.Error(r.Context(), "Error creating client", err)
				SendErrorWebsocket(ws, "Error creating client")
				return
			}
			clientID = createClientOutput.ClientID
		}

		wsBus := api.busFactory.NewBus(bus.WebSocketBusFactoryInput{
			ClientID: clientID,
			RoomID:   roomID,
			Socket:   ws,
		})

		output, err := api.usecases.JoinRoom.Execute(r.Context(), usecase.JoinRoomCommand{
			RoomID:   roomID,
			SenderID: clientID,
			Bus:      wsBus,
		})
		if err != nil {
			api.logger.Error(r.Context(), fmt.Sprintf("Error joining room %s", roomID), err)
			SendErrorWebsocket(ws, fmt.Sprintf("Error joining room %s", roomID))
			return
		}

		api.logger.Info(r.Context(), "New client connected: %v on room: %v", output.Client.ID, output.Room.ID)

		wsBus.Listen(r.Context())
	})
}
