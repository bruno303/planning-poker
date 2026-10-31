package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"planning-poker/internal/protocol"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type (
	WebSocketBusFactory struct {
		hub          domain.Hub
		usecases     usecase.UseCasesFacade
		ledger       CommandLedger
		websocketCfg WebSocketConfig
	}
	WebSocketBusFactoryInput struct {
		ClientID string
		RoomID   string
		Socket   *websocket.Conn
	}
	WebSocketMessage struct {
		Type      string `json:"type"`
		CommandID string `json:"commandId,omitempty"`
		Payload   any    `json:"payload"`
	}
	UpdateNamePayload struct {
		Username string `json:"username"`
	}
	ToggleSpectatorPayload struct {
		TargetClientID      string  `json:"targetClientId"`
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	ToggleOwnerPayload struct {
		TargetClientID      string  `json:"targetClientId"`
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	UpdateStoryPayload struct {
		Story               string  `json:"story"`
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	VotePayload struct {
		Vote string `json:"vote"`
	}
	AddStoryPayload struct {
		Story string `json:"story"`
	}
	RemoveStoryPayload struct {
		StoryID             string  `json:"storyId"`
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	SelectStoryPayload struct {
		StoryID             string  `json:"storyId"`
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	ReorderStoryPayload struct {
		StoryID             string  `json:"storyId"`
		TargetIndex         int     `json:"targetIndex"`
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	useCaseCall func(context.Context, WebSocketMessage) error

	WebsocketBus struct {
		ID          string
		conn        *websocket.Conn
		hub         domain.Hub
		logger      log.Logger
		cfg         WebSocketConfig
		calls       map[string]useCaseCall
		usecases    usecase.UseCasesFacade
		ledger      CommandLedger
		roomID      string
		closed      atomic.Bool
		closeOnce   sync.Once
		writeMu     sync.Mutex // Protects writes to conn (required by gorilla/websocket)
		done        chan struct{}
		skipCleanup atomic.Bool
	}

	WebSocketConfig struct {
		WriteTimeout time.Duration
		ReadTimeout  time.Duration
		PingInterval time.Duration
	}
)

var _ domain.Bus = (*WebsocketBus)(nil)

func NewWebSocketBusFactory(hub domain.Hub, usecases usecase.UseCasesFacade, ledger CommandLedger, websocketCfg WebSocketConfig) *WebSocketBusFactory {
	return &WebSocketBusFactory{
		hub:          hub,
		usecases:     usecases,
		ledger:       ledger,
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
		f.ledger,
		f.websocketCfg,
	)
}

func NewWebsocketBus(
	id string,
	roomID string,
	socket *websocket.Conn,
	hub domain.Hub,
	usecases usecase.UseCasesFacade,
	ledger CommandLedger,
	websocketCfg WebSocketConfig,
) *WebsocketBus {
	return &WebsocketBus{
		ID:       id,
		conn:     socket,
		hub:      hub,
		cfg:      websocketCfg,
		logger:   log.NewLogger("websocket.client"),
		usecases: usecases,
		ledger:   ledger,
		calls:    mapUsecases(usecases, id, roomID),
		roomID:   roomID,
		done:     make(chan struct{}),
	}
}

func (c *WebsocketBus) RoomID() string {
	return c.roomID
}

func (c *WebsocketBus) Detach() {
	c.skipCleanup.Store(true)
}

func (c *WebsocketBus) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.done)
		if !c.skipCleanup.Load() {
			err = c.leaveRoom(context.Background())
		}
		err2 := c.conn.Close()
		err = errors.Join(err, err2)
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

// sendRaw writes pre-encoded bytes using the same write serialization and
// deadline discipline as Send.
func (c *WebsocketBus) sendRaw(ctx context.Context, payload []byte) error {
	_, err := trace.Trace(ctx, trace.NameConfig("WebsocketBus", "sendRaw"), func(ctx context.Context) (any, error) {
		if c.closed.Load() {
			c.logger.Warn(ctx, "Attempted to send message to closed connection for client %v", c.ID)
			return nil, errors.New("connection closed")
		}
		c.logger.Debug(ctx, "Sending raw message to client: %v", payload)
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
		err := c.conn.WriteMessage(websocket.TextMessage, payload)
		if err != nil {
			c.logger.Error(ctx, fmt.Sprintf("WriteMessage error for client %v: %v", c.ID, err), err)
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
		c.processUnknown(ctx, msg)
		return
	}

	if msg.CommandID == "" {
		c.processLegacy(ctx, usecaseCall, msg)
		return
	}

	c.processCommand(ctx, usecaseCall, msg)
}

func (c *WebsocketBus) processUnknown(ctx context.Context, msg WebSocketMessage) {
	if msg.CommandID != "" {
		_ = c.Send(ctx, newCommandError(msg.CommandID, protocol.CommandErrorCodeUnknownCommand, "unknown command type", nil))
		return
	}
	c.logger.Error(ctx, fmt.Sprintf("Unknown event type '%v' for client %v", msg.Type, c.ID), errors.New("unknown event type"))
}

func (c *WebsocketBus) processCommand(ctx context.Context, usecaseCall useCaseCall, msg WebSocketMessage) {
	parsed, err := uuid.Parse(msg.CommandID)
	if err != nil {
		_ = c.Send(ctx, newCommandError(msg.CommandID, protocol.CommandErrorCodeInvalidCommandID, "commandId must be a UUID", nil))
		return
	}
	commandID := parsed.String()

	if c.ledger == nil {
		c.logger.Error(ctx, fmt.Sprintf("Command ledger unavailable for client %v", c.ID), errors.New("command ledger is not configured"))
		_ = c.Send(ctx, newCommandError(commandID, protocol.CommandErrorCodeInternal, "internal error", nil))
		return
	}

	outcome, found, err := c.ledger.Claim(ctx, c.roomID, c.ID, commandID)
	if err != nil {
		c.handleClaimError(ctx, commandID, err)
		return
	}

	if found {
		c.logger.Debug(ctx, "Replaying stored outcome for command %v of client %v", commandID, c.ID)
		_ = c.sendRaw(ctx, outcome)
		return
	}

	if err := usecaseCall(ctx, msg); err != nil {
		c.finishError(ctx, commandID, err)
		return
	}

	c.finishAck(ctx, commandID)
}

func (c *WebsocketBus) handleClaimError(ctx context.Context, commandID string, err error) {
	if errors.Is(err, ErrCommandInProgress) {
		c.logger.Debug(ctx, "Command %v for client %v is already being processed", commandID, c.ID)
		// protocol.CommandErrorCodeCommandInProgress is emitted when a commandId is
		// retried while a previous execution still holds its reservation.
		_ = c.Send(ctx, newCommandError(commandID, protocol.CommandErrorCodeCommandInProgress, "command is already being processed", nil))
		return
	}
	c.logger.Error(ctx, fmt.Sprintf("Failed to claim command %v for client %v", commandID, c.ID), err)
	_ = c.Send(ctx, newCommandError(commandID, protocol.CommandErrorCodeInternal, "internal error", nil))
}

func (c *WebsocketBus) finishError(ctx context.Context, commandID string, err error) {
	frame := commandErrorFor(commandID, err)
	if frame.Code == protocol.CommandErrorCodeStaleVersion {
		version, loadErr := c.roomVersion(ctx)
		if loadErr != nil {
			c.logger.Debug(ctx, "Failed to load room %v while reporting stale command for client %v: %v", c.roomID, c.ID, loadErr)
		} else {
			frame.RoomVersion = version
		}
	}
	if frame.Code == protocol.CommandErrorCodeInternal {
		c.logger.Error(ctx, fmt.Sprintf("Error handling event for client %v", c.ID), err)
	} else {
		c.logger.Debug(ctx, "Command %v failed for client %v with code %v: %v", commandID, c.ID, frame.Code, err)
	}
	c.finishCommand(ctx, commandID, frame)
}

func (c *WebsocketBus) finishAck(ctx context.Context, commandID string) {
	frame := newCommandAck(commandID, nil)
	version, err := c.roomVersion(ctx)
	if err != nil {
		c.logger.Debug(ctx, "Failed to load room %v while acknowledging command for client %v: %v", c.roomID, c.ID, err)
	} else {
		frame.RoomVersion = version
	}
	c.finishCommand(ctx, commandID, frame)
}

func (c *WebsocketBus) roomVersion(ctx context.Context) (*uint64, error) {
	room, err := c.hub.LoadRoom(ctx, c.roomID)
	if err != nil {
		return nil, err
	}
	version := room.RoomVersion
	return &version, nil
}

// finishCommand stores the encoded frame as the command outcome and writes those
// exact bytes back to the client.
func (c *WebsocketBus) finishCommand(ctx context.Context, commandID string, frame any) {
	payload, err := json.Marshal(frame)
	if err != nil {
		c.logger.Error(ctx, fmt.Sprintf("Failed to encode command frame for client %v", c.ID), err)
		_ = c.Send(ctx, frame)
		return
	}

	if err := c.ledger.Complete(ctx, c.roomID, c.ID, commandID, payload); err != nil {
		c.logger.Error(ctx, fmt.Sprintf("Failed to store outcome of command %v for client %v", commandID, c.ID), err)
	}

	_ = c.sendRaw(ctx, payload)
}

func (c *WebsocketBus) processLegacy(ctx context.Context, usecaseCall useCaseCall, msg WebSocketMessage) {
	if err := usecaseCall(ctx, msg); err != nil {
		if errors.Is(err, domain.ErrStaleRoomVersion) {
			room, loadErr := c.hub.LoadRoom(ctx, c.roomID)
			version := uint64(0)
			if loadErr == nil {
				version = room.RoomVersion
			}
			_ = c.Send(ctx, protocol.StaleCommand{Type: protocol.MessageTypeStaleCommand, RoomVersion: version})
			return
		}
		c.logger.Error(ctx, fmt.Sprintf("Error handling event for client %v", c.ID), err)
	}
}

func expectedRoomVersion(payload any) (*uint64, bool, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, false, ErrInvalidPayload
	}
	var value struct {
		ExpectedRoomVersion *uint64 `json:"expectedRoomVersion"`
	}
	if json.Unmarshal(data, &value) != nil {
		return nil, false, ErrInvalidPayload
	}
	if value.ExpectedRoomVersion == nil {
		return nil, false, nil
	}
	return value.ExpectedRoomVersion, true, nil
}

func withExpectedRoomVersion(payload any, execute func(*uint64) error) error {
	expected, _, err := expectedRoomVersion(payload)
	if err != nil {
		return err
	}
	return execute(expected)
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

func decodePayload(payload, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, out)
}

func withPayload[T any](payload any, execute func(T) error) error {
	var decoded T
	if err := decodePayload(payload, &decoded); err != nil {
		return ErrInvalidPayload
	}

	return execute(decoded)
}

func mapUsecases(usecases usecase.UseCasesFacade, clientID, roomID string) map[string]useCaseCall {

	return map[string]useCaseCall{
		"update-name": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload UpdateNamePayload) error {
				return usecases.UpdateName.Execute(ctx, usecase.UpdateNameCommand{
					RoomID:   roomID,
					SenderID: clientID,
					Username: payload.Username,
				})
			})
		},
		"vote": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload VotePayload) error {
				return usecases.Vote.Execute(ctx, usecase.VoteCommand{
					RoomID:   roomID,
					SenderID: clientID,
					Vote:     lo.ToPtr(payload.Vote),
				})
			})
		},
		"reset": func(ctx context.Context, msg WebSocketMessage) error {
			return withExpectedRoomVersion(msg.Payload, func(expected *uint64) error {
				return usecases.Reset.Execute(ctx, usecase.ResetCommand{RoomID: roomID, SenderID: clientID, ExpectedRoomVersion: expected})
			})
		},
		"reveal-votes": func(ctx context.Context, msg WebSocketMessage) error {
			return withExpectedRoomVersion(msg.Payload, func(expected *uint64) error {
				return usecases.Reveal.Execute(ctx, usecase.RevealCommand{RoomID: roomID, SenderID: clientID, ExpectedRoomVersion: expected})
			})
		},
		"toggle-spectator": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload ToggleSpectatorPayload) error {
				return usecases.ToggleSpectator.Execute(ctx, usecase.ToggleSpectatorCommand{
					RoomID:              roomID,
					SenderID:            clientID,
					TargetClientID:      payload.TargetClientID,
					ExpectedRoomVersion: payload.ExpectedRoomVersion,
				})
			})
		},
		"toggle-owner": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload ToggleOwnerPayload) error {
				return usecases.ToggleOwner.Execute(ctx, usecase.ToggleOwnerCommand{
					RoomID:              roomID,
					SenderID:            clientID,
					TargetClientID:      payload.TargetClientID,
					ExpectedRoomVersion: payload.ExpectedRoomVersion,
				})
			})
		},
		"update-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload UpdateStoryPayload) error {
				return usecases.UpdateStory.Execute(ctx, usecase.UpdateStoryCommand{
					RoomID:              roomID,
					SenderID:            clientID,
					Story:               payload.Story,
					ExpectedRoomVersion: payload.ExpectedRoomVersion,
				})
			})
		},
		"vote-again": func(ctx context.Context, msg WebSocketMessage) error {
			return withExpectedRoomVersion(msg.Payload, func(expected *uint64) error {
				return usecases.VoteAgain.Execute(ctx, usecase.VoteAgainCommand{RoomID: roomID, SenderID: clientID, ExpectedRoomVersion: expected})
			})
		},
		"toggle-backlog-mode": func(ctx context.Context, msg WebSocketMessage) error {
			return withExpectedRoomVersion(msg.Payload, func(expected *uint64) error {
				return usecases.ToggleBacklogMode.Execute(ctx, usecase.ToggleBacklogModeCommand{RoomID: roomID, SenderID: clientID, ExpectedRoomVersion: expected})
			})
		},
		"add-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload AddStoryPayload) error {
				return usecases.AddStory.Execute(ctx, usecase.AddStoryCommand{
					RoomID:    roomID,
					SenderID:  clientID,
					StoryName: payload.Story,
				})
			})
		},
		"remove-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload RemoveStoryPayload) error {
				return usecases.RemoveStory.Execute(ctx, usecase.RemoveStoryCommand{
					RoomID:              roomID,
					SenderID:            clientID,
					StoryID:             payload.StoryID,
					ExpectedRoomVersion: payload.ExpectedRoomVersion,
				})
			})
		},
		"advance-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withExpectedRoomVersion(msg.Payload, func(expected *uint64) error {
				return usecases.AdvanceStory.Execute(ctx, usecase.AdvanceStoryCommand{RoomID: roomID, SenderID: clientID, ExpectedRoomVersion: expected})
			})
		},
		"prev-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withExpectedRoomVersion(msg.Payload, func(expected *uint64) error {
				return usecases.PrevStory.Execute(ctx, usecase.PrevStoryCommand{RoomID: roomID, SenderID: clientID, ExpectedRoomVersion: expected})
			})
		},
		"select-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload SelectStoryPayload) error {
				return usecases.SelectStory.Execute(ctx, usecase.SelectStoryCommand{
					RoomID:              roomID,
					SenderID:            clientID,
					StoryID:             payload.StoryID,
					ExpectedRoomVersion: payload.ExpectedRoomVersion,
				})
			})
		},
		"reorder-story": func(ctx context.Context, msg WebSocketMessage) error {
			return withPayload(msg.Payload, func(payload ReorderStoryPayload) error {
				return usecases.ReorderStory.Execute(ctx, usecase.ReorderStoryCommand{
					RoomID:              roomID,
					SenderID:            clientID,
					StoryID:             payload.StoryID,
					TargetIndex:         payload.TargetIndex,
					ExpectedRoomVersion: payload.ExpectedRoomVersion,
				})
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
