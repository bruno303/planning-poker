package bus

import (
	"context"
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
	useCaseCall  func(context.Context, map[string]any) error
	useCaseCall2 func(context.Context, WebSocketMessage) error

	WebsocketBus struct {
		ID        string
		conn      *websocket.Conn
		hub       domain.Hub
		logger    log.Logger
		cfg       WebSocketConfig
		calls     map[string]useCaseCall
		calls2    map[string]useCaseCall2
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
		calls2:   mapUsecases2(usecases, id, roomID),
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
		return nil, c.conn.WriteJSON(message)
	})
	return err
}

func (c *WebsocketBus) receive(ctx context.Context) (map[string]any, error) {
	msg, err := trace.Trace(ctx, trace.NameConfig("WebsocketBus", "receive"), func(ctx context.Context) (any, error) {
		var msg map[string]any
		err := c.conn.ReadJSON(&msg)
		return msg, err
	})
	if err != nil {
		return nil, err
	}
	return msg.(map[string]any), nil
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

		eventType, ok := msg["type"].(string)
		if !ok {
			c.logger.Error(ctx, fmt.Sprintf("Error casting message type to string for client %v", c.ID), errors.New("invalid message format"))
			continue
		}

		if msg["payload"] != nil {
			c.processAsMessage(ctx, eventType, msg)
			continue
		}

		c.processAsMap(ctx, eventType, msg)
	}
}

func (c *WebsocketBus) processAsMap(ctx context.Context, eventType string, msg map[string]any) {
	c.logger.Debug(ctx, "Processing message as map for event type '%v' with payload: %v", eventType, msg)
	usecaseCall, exists := c.calls[eventType]
	if !exists {
		c.logger.Error(ctx, fmt.Sprintf("Unknown event type '%v' for client %v", eventType, c.ID), errors.New("unknown event type"))
		return
	}

	err := usecaseCall(ctx, msg)
	if err != nil {
		c.logger.Error(ctx, fmt.Sprintf("Error handling event for client %v", c.ID), err)
		return
	}
}

func (c *WebsocketBus) processAsMessage(ctx context.Context, eventType string, msg map[string]any) {
	c.logger.Debug(ctx, "Processing message as WebSocketMessage for event type '%v' with payload: %v", eventType, msg)
	usecaseCall, exists := c.calls2[eventType]
	if !exists {
		c.logger.Error(ctx, fmt.Sprintf("Unknown event type '%v' for client %v", eventType, c.ID), errors.New("unknown event type"))
		return
	}

	err := usecaseCall(ctx, WebSocketMessage{
		Type:    eventType,
		Payload: c.buildPayload(eventType, msg["payload"].(map[string]any)),
	})
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
	return map[string]useCaseCall{
		"update-name": func(ctx context.Context, msg map[string]any) error {
			return usecases.UpdateName.Execute(ctx, usecase.UpdateNameCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Username: msg["username"].(string),
			})
		},
		"vote": func(ctx context.Context, msg map[string]any) error {
			return usecases.Vote.Execute(ctx, usecase.VoteCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Vote:     lo.ToPtr(msg["vote"].(string)),
			})
		},
		"reset": func(ctx context.Context, msg map[string]any) error {
			return usecases.Reset.Execute(ctx, usecase.ResetCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
		"reveal-votes": func(ctx context.Context, msg map[string]any) error {
			return usecases.Reveal.Execute(ctx, usecase.RevealCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
		"toggle-spectator": func(ctx context.Context, msg map[string]any) error {
			return usecases.ToggleSpectator.Execute(ctx, usecase.ToggleSpectatorCommand{
				RoomID:         roomID,
				SenderID:       clientID,
				TargetClientID: msg["targetClientId"].(string),
			})
		},
		"toggle-owner": func(ctx context.Context, msg map[string]any) error {
			return usecases.ToggleOwner.Execute(ctx, usecase.ToggleOwnerCommand{
				RoomID:         roomID,
				SenderID:       clientID,
				TargetClientID: msg["targetClientId"].(string),
			})
		},
		"update-story": func(ctx context.Context, msg map[string]any) error {
			return usecases.UpdateStory.Execute(ctx, usecase.UpdateStoryCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Story:    msg["story"].(string),
			})
		},
		"new-voting": func(ctx context.Context, msg map[string]any) error {
			return usecases.NewVoting.Execute(ctx, usecase.NewVotingCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
		"vote-again": func(ctx context.Context, msg map[string]any) error {
			return usecases.VoteAgain.Execute(ctx, usecase.VoteAgainCommand{
				RoomID:   roomID,
				SenderID: clientID,
			})
		},
	}
}

func mapUsecases2(usecases usecase.UseCasesFacade, clientID, roomID string) map[string]useCaseCall2 {
	return map[string]useCaseCall2{
		"update-name": func(ctx context.Context, msg WebSocketMessage) error {
			payload, ok := msg.Payload.(UpdateNamePayload)
			if !ok {
				return errors.New("invalid payload")
			}
			return usecases.UpdateName.Execute(ctx, usecase.UpdateNameCommand{
				RoomID:   roomID,
				SenderID: clientID,
				Username: payload.Username,
			})
		},
		"vote": func(ctx context.Context, msg WebSocketMessage) error {
			payload, ok := msg.Payload.(VotePayload)
			if !ok {
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
			payload, ok := msg.Payload.(ToggleSpectatorPayload)
			if !ok {
				return errors.New("invalid payload")
			}
			return usecases.ToggleSpectator.Execute(ctx, usecase.ToggleSpectatorCommand{
				RoomID:         roomID,
				SenderID:       clientID,
				TargetClientID: payload.TargetClientID,
			})
		},
		"toggle-owner": func(ctx context.Context, msg WebSocketMessage) error {
			payload, ok := msg.Payload.(ToggleOwnerPayload)
			if !ok {
				return errors.New("invalid payload")
			}
			return usecases.ToggleOwner.Execute(ctx, usecase.ToggleOwnerCommand{
				RoomID:         roomID,
				SenderID:       clientID,
				TargetClientID: payload.TargetClientID,
			})
		},
		"update-story": func(ctx context.Context, msg WebSocketMessage) error {
			payload, ok := msg.Payload.(UpdateStoryPayload)
			if !ok {
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

func (c *WebsocketBus) buildPayload(eventType string, payload map[string]any) any {
	switch eventType {
	case "update-name":
		return UpdateNamePayload{
			Username: payload["username"].(string),
		}
	case "vote":
		return VotePayload{
			Vote: payload["vote"].(string),
		}
	case "toggle-spectator":
		return ToggleSpectatorPayload{
			TargetClientID: payload["targetClientId"].(string),
		}
	case "toggle-owner":
		return ToggleOwnerPayload{
			TargetClientID: payload["targetClientId"].(string),
		}
	case "update-story":
		return UpdateStoryPayload{
			Story: payload["story"].(string),
		}
	default:
		return nil
	}
}

func (c *WebsocketBus) leaveRoom(ctx context.Context) error {
	return c.usecases.LeaveRoom.Execute(ctx, usecase.LeaveRoomCommand{
		RoomID:   c.roomID,
		SenderID: c.ID,
	})
}
