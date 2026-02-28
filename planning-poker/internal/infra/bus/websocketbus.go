package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type (
	WebSocketBusFactory struct {
		hub          domain.Hub
		usecases     usecase.UseCasesFacade
		websocketCfg WebSocketConfig
	}
	WebSocketBusFactoryInput struct {
		ClientID string
		RoomID   string
		Socket   *websocket.Conn
	}
	WebSocketMessage struct {
		Type    string `json:"type"`
		Payload any    `json:"payload"`
	}
	UpdateNamePayload struct {
		Username string `json:"username"`
	}
	ToggleSpectatorPayload struct {
		TargetClientID string `json:"targetClientId"`
	}
	ToggleOwnerPayload struct {
		TargetClientID string `json:"targetClientId"`
	}
	UpdateStoryPayload struct {
		Story string `json:"story"`
	}
	VotePayload struct {
		Vote string `json:"vote"`
	}
	useCaseCall func(context.Context, WebSocketMessage) error

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

var _ domain.Bus = (*WebsocketBus)(nil)

func NewWebSocketBusFactory(hub domain.Hub, usecases usecase.UseCasesFacade, websocketCfg WebSocketConfig) *WebSocketBusFactory {
	return &WebSocketBusFactory{
		hub:          hub,
		usecases:     usecases,
		websocketCfg: websocketCfg,
	}
}

func (f *WebSocketBusFactory) NewBus(input WebSocketBusFactoryInput) domain.Bus {
	return NewWebsocketBus(
		input.ClientID,
		input.RoomID,
		input.Socket,
		f.hub,
		f.usecases,
		f.websocketCfg,
	)
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
		calls:    mapUsecases(usecases, id, roomID),
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
	_, err := trace.Trace(ctx, trace.NameConfig("WebsocketBus", "send"), func(ctx context.Context) (any, error) {
		if c.closed.Load() {
			c.logger.Warn(ctx, "Attempted to send message to closed connection for client %v", c.ID)
			return nil, errors.New("connection closed")
		}
		c.logger.Debug(ctx, "Sending message to client: %v", message)
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
		err := c.conn.WriteJSON(message)
		if err != nil {
			c.logger.Error(ctx, fmt.Sprintf("WriteJSON error for client %v: %v", c.ID, err), err)
		}
		return nil, err
	})
	return err
}

func (c *WebsocketBus) receive(ctx context.Context) (WebSocketMessage, error) {
	msg, err := trace.Trace(ctx, trace.NameConfig("WebsocketBus", "receive"), func(ctx context.Context) (any, error) {
		var msg WebSocketMessage
		err := c.conn.ReadJSON(&msg)
		return msg, err
	})
	if err != nil {
		return WebSocketMessage{}, err
	}
	return msg.(WebSocketMessage), nil
}

func (c *WebsocketBus) Listen(ctx context.Context) {
	defer func() { _ = c.Close() }()

	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))
	go c.pinger(ctx)

	for {
		msg, err := c.receive(ctx)
		_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout))

		if err != nil {
			c.handleReceiveError(ctx, err)
			return
		}
		c.logger.Info(ctx, "Message received from client %v: %v", c.ID, msg)

		c.process(ctx, msg)
	}
}

func (c *WebsocketBus) process(ctx context.Context, msg WebSocketMessage) {
	c.logger.Debug(ctx, "Processing message for event type '%v' with payload: %v", msg.Type, msg)
	usecaseCall, exists := c.calls[msg.Type]
	if !exists {
		c.logger.Error(ctx, fmt.Sprintf("Unknown event type '%v' for client %v", msg.Type, c.ID), errors.New("unknown event type"))
		return
	}

	err := usecaseCall(ctx, msg)
	if err != nil {
		c.logger.Error(ctx, fmt.Sprintf("Error handling event for client %v", c.ID), err)
		return
	}
}

func (c *WebsocketBus) handleReceiveError(ctx context.Context, err error) {
	_, _ = trace.Trace(ctx, trace.NameConfig("WebsocketBus", "handleReceiveError"), func(ctx context.Context) (any, error) {
		// closed connection
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			c.logger.Warn(ctx, "Reader: Connection explicitly closed by client or network: %v", err.Error())
			return nil, nil
		}

		// I/O timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			c.logger.Error(ctx, "WebSocket Timeout Detected! Client failed to respond to Ping: %v", netErr)
			return nil, nil
		}

		// general errors
		c.logger.Error(ctx, fmt.Sprintf("Error receiving message from client %v", c.ID), err)
		return nil, nil
	})
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
			_, err := trace.Trace(ctx, trace.NameConfig("WebsocketBus", "pinger"), func(ctx context.Context) (any, error) {
				if c.closed.Load() {
					return nil, nil
				}
				c.writeMu.Lock()
				err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
				if err == nil {
					err = c.conn.WriteMessage(websocket.PingMessage, nil)
				}
				c.writeMu.Unlock()
				return nil, err
			})
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

func mapUsecases(usecases usecase.UseCasesFacade, clientID, roomID string) map[string]useCaseCall {
	decode := func(payload any, out any) error {
		if v, ok := payload.(map[string]any); ok {
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			return json.Unmarshal(b, out)
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, out)
	}

	return map[string]useCaseCall{
		"update-name": func(ctx context.Context, msg WebSocketMessage) error {
			var payload UpdateNamePayload
			if err := decode(msg.Payload, &payload); err != nil {
				return errors.New("invalid payload")
			}
			return usecases.UpdateName.Execute(ctx, usecase.UpdateNameCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Username: payload.Username,
			})
		},
		"vote": func(ctx context.Context, msg WebSocketMessage) error {
			var payload VotePayload
			if err := decode(msg.Payload, &payload); err != nil {
				return errors.New("invalid payload")
			}
			return usecases.Vote.Execute(ctx, usecase.VoteCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Vote:     lo.ToPtr(payload.Vote),
			})
		},
		"reset": func(ctx context.Context, msg WebSocketMessage) error {
			return usecases.Reset.Execute(ctx, usecase.ResetCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
		"reveal-votes": func(ctx context.Context, msg WebSocketMessage) error {
			return usecases.Reveal.Execute(ctx, usecase.RevealCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
		"toggle-spectator": func(ctx context.Context, msg WebSocketMessage) error {
			var payload ToggleSpectatorPayload
			if err := decode(msg.Payload, &payload); err != nil {
				return errors.New("invalid payload")
			}
			return usecases.ToggleSpectator.Execute(ctx, usecase.ToggleSpectatorCommand{
				RoomID:         roomID,
				SenderID:       clientID,
				TargetClientID: payload.TargetClientID,
			})
		},
		"toggle-owner": func(ctx context.Context, msg WebSocketMessage) error {
			var payload ToggleOwnerPayload
			if err := decode(msg.Payload, &payload); err != nil {
				return errors.New("invalid payload")
			}
			return usecases.ToggleOwner.Execute(ctx, usecase.ToggleOwnerCommand{
				RoomID:         roomID,
				SenderID:       clientID,
				TargetClientID: payload.TargetClientID,
			})
		},
		"update-story": func(ctx context.Context, msg WebSocketMessage) error {
			var payload UpdateStoryPayload
			if err := decode(msg.Payload, &payload); err != nil {
				return errors.New("invalid payload")
			}
			return usecases.UpdateStory.Execute(ctx, usecase.UpdateStoryCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Story:    payload.Story,
			})
		},
		"new-voting": func(ctx context.Context, msg WebSocketMessage) error {
			return usecases.NewVoting.Execute(ctx, usecase.NewVotingCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
		"vote-again": func(ctx context.Context, msg WebSocketMessage) error {
			return usecases.VoteAgain.Execute(ctx, usecase.VoteAgainCommand{
				RoomID:   roomID,
				SenderID: clientID,
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
