package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type (
	contextKey  string
	useCaseCall func(context.Context, map[string]any) error

	WebsocketAPI struct {
		upgrader     websocket.Upgrader
		hub          domain.Hub
		usecases     usecase.UseCasesFacade
		websocketCfg WebSocketConfig
		logger       log.Logger
	}

	WebsocketBus struct {
		ID        string
		conn      *websocket.Conn
		hub       domain.Hub
		logger    log.Logger
		cfg       WebSocketConfig
		calls     map[string]useCaseCall
		usecases  usecase.UseCasesFacade
		roomID    string
		closed    atomic.Bool
		closeOnce sync.Once
		writeMu   sync.Mutex // Protects writes to conn (required by gorilla/websocket)
		done      chan struct{}
	}

	WebSocketConfig struct {
		WriteTimeout time.Duration
		ReadTimeout  time.Duration
		PingInterval time.Duration
	}
)

var _ API = (*WebsocketAPI)(nil)

func NewWebsocketAPI(hub domain.Hub, usecases usecase.UseCasesFacade, websocketCfg WebSocketConfig) *WebsocketAPI {
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
				return NewWebsocketBus(clientID, roomID, ws, api.hub, api.usecases, api.websocketCfg)
			},
		})
		if err != nil {
			api.logger.Error(r.Context(), fmt.Sprintf("Error joining room %s", roomID), err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, fmt.Sprintf("Error joining room %s", roomID))
			return
		}

		api.logger.Info(r.Context(), "New client connected: %v on room: %v", output.Client.ID, output.Room.ID)

		output.Bus.Listen(r.Context())
	})
}

func NewWebsocketBus(
	id string,
	roomID string,
	socket *websocket.Conn,
	hub domain.Hub,
	usecases usecase.UseCasesFacade,
	websocketCfg WebSocketConfig,
) *WebsocketBus {
	return &WebsocketBus{
		ID:       id,
		conn:     socket,
		hub:      hub,
		cfg:      websocketCfg,
		logger:   log.NewLogger("websocket.client"),
		usecases: usecases,
		calls:    mapUsecases(usecases),
		roomID:   roomID,
		done:     make(chan struct{}),
	}
}

func (c *WebsocketBus) RoomID() string {
	return c.roomID
}

func (c *WebsocketBus) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.done)
		err1 := c.leaveRoom(context.Background())
		err2 := c.conn.Close()
		err = errors.Join(err1, err2)
	})
	return err
}

func (c *WebsocketBus) Send(ctx context.Context, message any) error {
	if c.closed.Load() {
		c.logger.Warn(ctx, "Attempted to send message to closed connection for client %v", c.ID)
		return errors.New("connection closed")
	}
	c.logger.Debug(ctx, "Sending message to client: %v", message)
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
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
		_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))

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

		usecaseCall, exists := c.calls[eventType]
		if !exists {
			c.logger.Error(ctx, fmt.Sprintf("Unknown event type '%v' for client %v", eventType, c.ID), errors.New("unknown event type"))
			continue
		}

		err = usecaseCall(ctx, msg)
		if err != nil {
			c.logger.Error(ctx, fmt.Sprintf("Error handling event for client %v", c.ID), err)
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
			c.writeMu.Lock()
			err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
			if err == nil {
				err = c.conn.WriteMessage(websocket.PingMessage, nil)
			}
			c.writeMu.Unlock()

			if err != nil {
				c.logger.Error(ctx, "Error while pinging the client", err)
				return
			}
		case <-c.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func mapUsecases(usecases usecase.UseCasesFacade) map[string]useCaseCall {
	return map[string]useCaseCall{
		"update-name": func(ctx context.Context, msg map[string]any) error {
			return usecases.UpdateName.Execute(ctx, usecase.UpdateNameCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
				Username: msg["username"].(string),
			})
		},
		"vote": func(ctx context.Context, msg map[string]any) error {
			return usecases.Vote.Execute(ctx, usecase.VoteCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
				Vote:     lo.ToPtr(msg["vote"].(string)),
			})
		},
		"reset": func(ctx context.Context, msg map[string]any) error {
			return usecases.Reset.Execute(ctx, usecase.ResetCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		},
		"reveal-votes": func(ctx context.Context, msg map[string]any) error {
			return usecases.Reveal.Execute(ctx, usecase.RevealCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		},
		"toggle-spectator": func(ctx context.Context, msg map[string]any) error {
			return usecases.ToggleSpectator.Execute(ctx, usecase.ToggleSpectatorCommand{
				RoomID:         msg["roomId"].(string),
				SenderID:       msg["clientId"].(string),
				TargetClientID: msg["targetClientId"].(string),
			})
		},
		"toggle-owner": func(ctx context.Context, msg map[string]any) error {
			return usecases.ToggleOwner.Execute(ctx, usecase.ToggleOwnerCommand{
				RoomID:         msg["roomId"].(string),
				SenderID:       msg["clientId"].(string),
				TargetClientID: msg["targetClientId"].(string),
			})
		},
		"update-story": func(ctx context.Context, msg map[string]any) error {
			return usecases.UpdateStory.Execute(ctx, usecase.UpdateStoryCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
				Story:    msg["story"].(string),
			})
		},
		"new-voting": func(ctx context.Context, msg map[string]any) error {
			return usecases.NewVoting.Execute(ctx, usecase.NewVotingCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		},
		"vote-again": func(ctx context.Context, msg map[string]any) error {
			return usecases.VoteAgain.Execute(ctx, usecase.VoteAgainCommand{
				RoomID:   msg["roomId"].(string),
				SenderID: msg["clientId"].(string),
			})
		},
	}
}

func (c *WebsocketBus) leaveRoom(ctx context.Context) error {
	return c.usecases.LeaveRoom.Execute(ctx, usecase.LeaveRoomCommand{
		RoomID:   c.roomID,
		SenderID: c.ID,
	})
}
