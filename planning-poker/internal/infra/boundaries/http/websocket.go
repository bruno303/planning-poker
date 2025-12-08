package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type (
	contextKey string

	WebsocketAPI struct {
		upgrader     websocket.Upgrader
		hub          domain.Hub
		usecases     usecase.UseCases
		websocketCfg WebSocketConfig
		logger       log.Logger
	}

	WebsocketBus struct {
		ID       string
		conn     *websocket.Conn
		hub      domain.Hub
		usecases usecase.UseCases
		logger   log.Logger
		cfg      WebSocketConfig
	}

	WebSocketConfig struct {
		WriteTimeout time.Duration
		ReadTimeout  time.Duration
		PingInterval time.Duration
	}
)

var _ API = (*WebsocketAPI)(nil)

func NewWebsocketAPI(hub domain.Hub, usecases usecase.UseCases, websocketCfg WebSocketConfig) *WebsocketAPI {
	return &WebsocketAPI{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		hub:          hub,
		usecases:     usecases,
		websocketCfg: websocketCfg,
		logger:       log.NewLogger("planningpoker.api"),
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

		output, err := api.usecases.JoinRoom.Execute(r.Context(), usecase.JoinRoomCommand{
			RoomID:   roomID,
			SenderID: uuid.NewString(),
			BusFactory: func(clientID string) domain.Bus {
				return NewWebsocketBus(clientID, ws, api.hub, api.usecases, api.websocketCfg)
			},
		})
		if err != nil {
			api.logger.Error(r.Context(), fmt.Sprintf("Error joining room %s", roomID), err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, fmt.Sprintf("Error joining room %s", roomID))
			return
		}

		api.logger.Info(r.Context(), "New client connected: %v on room: %v", output.Client.ID, output.Room.ID)

		defer func() { _ = output.Bus.Close() }()
		defer func() {
			err := api.usecases.LeaveRoom.Execute(r.Context(), usecase.LeaveRoomCommand{
				RoomID:   output.Room.ID,
				SenderID: output.Client.ID,
			})
			if err != nil {
				api.logger.Error(r.Context(), fmt.Sprintf("Error leaving room %s", roomID), err)
			}
		}()

		output.Bus.Listen(r.Context())
	})
}

func NewWebsocketBus(
	id string,
	socket *websocket.Conn,
	hub domain.Hub,
	usecases usecase.UseCases,
	websocketCfg WebSocketConfig,
) *WebsocketBus {
	return &WebsocketBus{
		ID:       id,
		conn:     socket,
		hub:      hub,
		usecases: usecases,
		cfg:      websocketCfg,
		logger:   log.NewLogger("websocket.client"),
	}
}

func (c *WebsocketBus) Close() error {
	return c.conn.Close()
}

func (c *WebsocketBus) Send(ctx context.Context, message any) error {
	c.logger.Debug(ctx, "Sending message to client: %v", message)
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
	return c.conn.WriteJSON(message)
}

func (c *WebsocketBus) receive() (map[string]any, error) {
	var msg map[string]any
	err := c.conn.ReadJSON(&msg)
	return msg, err
}

func (c *WebsocketBus) Listen(ctx context.Context) {
	defer func() { _ = c.Close() }()

	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
	go c.pinger(ctx)

	for {
		msg, err := c.receive()
		if err != nil {
			c.handleReceiveError(ctx, err)

			return
		}
		c.logger.Info(ctx, "Message received from client %v: %v", c.ID, msg)

		eventType, ok := msg["type"].(string)
		if !ok {
			c.logger.Error(ctx, fmt.Sprintf("Error casting message type to string for client %v", c.ID), err)
			continue
		}

		var uerr error
		switch eventType {
		case "update-name":
			uerr = c.usecases.UpdateName.Execute(ctx, usecase.UpdateNameCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
				Username: msg["username"].(string),
			})
		case "vote":
			uerr = c.usecases.Vote.Execute(ctx, usecase.VoteCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
				Vote:     lo.ToPtr(msg["vote"].(string)),
			})
		case "reset":
			uerr = c.usecases.Reset.Execute(ctx, usecase.ResetCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		case "reveal-votes":
			uerr = c.usecases.Reveal.Execute(ctx, usecase.RevealCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		case "toggle-spectator":
			uerr = c.usecases.ToggleSpectator.Execute(ctx, usecase.ToggleSpectatorCommand{
				RoomID:         msg["roomId"].(string),
				SenderID:       msg["clientId"].(string),
				TargetClientID: msg["targetClientId"].(string),
			})
		case "toggle-owner":
			uerr = c.usecases.ToggleOwner.Execute(ctx, usecase.ToggleOwnerCommand{
				RoomID:         msg["roomId"].(string),
				SenderID:       msg["clientId"].(string),
				TargetClientID: msg["targetClientId"].(string),
			})
		case "update-story":
			uerr = c.usecases.UpdateStory.Execute(ctx, usecase.UpdateStoryCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
				Story:    msg["story"].(string),
			})
		case "new-voting":
			uerr = c.usecases.NewVoting.Execute(ctx, usecase.NewVotingCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		case "vote-again":
			uerr = c.usecases.VoteAgain.Execute(ctx, usecase.VoteAgainCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		default:
			c.logger.Error(ctx, fmt.Sprintf("Unknown event type '%v' for client %v", eventType, c.ID), errors.New("unknown event type"))
			continue
		}

		if uerr != nil {
			c.logger.Error(ctx, fmt.Sprintf("Error handling event for client %v", c.ID), uerr)
			continue
		}
	}
}

func (c *WebsocketBus) handleReceiveError(ctx context.Context, err error) {
	// closed connection
	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		c.logger.Warn(ctx, "Reader: Connection explicitly closed by client or network: %v", err.Error())
		return
	}

	// I/O timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		c.logger.Error(ctx, "WebSocket Timeout Detected! Client failed to respond to Ping: %v", netErr)
		return
	}

	// general errors
	c.logger.Error(ctx, fmt.Sprintf("Error receiving message from client %v", c.ID), err)
}

func (c *WebsocketBus) pinger(ctx context.Context) {
	c.conn.SetPongHandler(func(appData string) error {
		c.logger.Debug(ctx, "Pong received from client %v", c.ID)
		_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
		return nil
	})

	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout)); err != nil {
				c.logger.Error(ctx, "Error while setting the write deadline", err)
				return
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error(ctx, "Error while pinging the client", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
