package entity

import (
	"context"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	Client struct {
		ID   string
		Name string

		CurrentVote *Vote
		HasVoted    bool
		VotedAt     *time.Time
		IsSpectator bool
		IsOwner     bool

		logger log.Logger
	}
)

func newClient(id string) *Client {
	return &Client{
		ID:     id,
		logger: log.NewLogger("planningpoker.client"),
	}
}

func (c *Client) WithLogger(logger log.Logger) *Client {
	c.logger = logger
	return c
}

func (c *Client) UpdateName(ctx context.Context, name string) {
	c.Name = name
}
