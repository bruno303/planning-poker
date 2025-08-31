package planningpoker

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	conn *websocket.Conn
	room *Room
}

func NewClient(id string, conn *websocket.Conn, room *Room) *Client {
	return &Client{
		ID:   id,
		conn: conn,
		room: room,
	}
}
func (c *Client) Close() error {
	return c.conn.Close()
}
func (c *Client) Send(message any) error {
	return c.conn.WriteJSON(message)
}
func (c *Client) Receive() (map[string]any, error) {
	var msg map[string]any
	err := c.conn.ReadJSON(&msg)
	return msg, err
}

func (c *Client) Listen(ctx context.Context) {
	for {
		msg, err := c.Receive()
		if err != nil {
			logger.Error(ctx, fmt.Sprintf("Error receiving message from client %v", c.ID), err)
			return
		}
		logger.Info(ctx, "Message received from client %v: %v", c.ID, msg)
		c.room.Broadcast(msg)
	}
}
