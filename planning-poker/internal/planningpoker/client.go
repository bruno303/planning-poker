package planningpoker

import (
	"context"
	"fmt"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type Client struct {
	ID   string
	Name string

	Vote        *string
	HasVoted    bool
	IsSpectator bool
	IsOwner     bool

	conn *websocket.Conn
	room *Room
}

func NewClient(conn *websocket.Conn, room *Room) *Client {
	return &Client{
		ID:   uuid.NewString(),
		conn: conn,
		room: room,
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Send(ctx context.Context, message any) error {
	log.Log().Debug(ctx, "Sending message to client %v: %v", c.ID, message)
	return c.conn.WriteJSON(message)
}

func (c *Client) receive() (map[string]any, error) {
	var msg map[string]any
	err := c.conn.ReadJSON(&msg)
	return msg, err
}

func (c *Client) vote(ctx context.Context, vote *string) {
	if c.room.Reveal {
		logger.Debug(ctx, "Vote ignored for client %s because votes are already revealed", c.ID)
		return
	}

	c.Vote = vote
	if vote != nil && *vote != "" {
		c.HasVoted = true
	} else {
		c.HasVoted = false
	}

	c.room.checkReveal()
}

func (c *Client) updateName(ctx context.Context, name string) {
	c.Name = name
	c.Send(ctx, NewUpdateClientIDCommand(c.ID))
}

func (c *Client) Listen(ctx context.Context) {
	defer c.Close()
	defer func() {
		c.room.RemoveClient(ctx, c.ID)
		if len(c.room.Clients) == 0 {
			c.room.Hub.RemoveRoom(c.room.ID)
			logger.Info(ctx, "Room '%s' removed from hub", c.room.ID)
		}
	}()

	for {
		msg, err := c.receive()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				logger.Info(ctx, "Client %v disconnected", c.ID)

			} else {
				logger.Error(ctx, fmt.Sprintf("Error receiving message from client %v", c.ID), err)
			}

			return
		}
		logger.Info(ctx, "Message received from client %v: %v", c.ID, msg)

		switch msg["type"] {
		case "init":
			c.updateName(ctx, msg["username"].(string))

		case "vote":
			c.vote(ctx, lo.ToPtr(msg["vote"].(string)))

		case "toggle-spectator":
			c.executeIfOwner(func() {
				c.room.ToggleSpectator(ctx, msg["id"].(string))
			})

		case "toggle-owner":
			c.executeIfOwner(func() {
				c.room.ToggleOwner(msg["id"].(string))
			})

		case "new-voting":
			c.executeIfOwner(func() {
				c.room.NewVoting()
			})

		case "reveal-votes":
			c.executeIfOwner(func() {
				c.room.Reveal = !c.room.Reveal
			})

		case "update-story":
			c.executeIfOwner(func() {
				c.room.SetCurrentStory(msg["story"].(string))
			})

		case "vote-again":
			c.executeIfOwner(func() {
				c.room.ResetVoting()
			})
		}

		// always broadcast the current state
		c.room.Broadcast(ctx, NewRoomStateCommand(c.room))
	}
}

func (c *Client) executeIfOwner(fn func()) {
	if c.IsOwner {
		fn()
	}
}
