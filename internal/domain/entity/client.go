package entity

import (
	"context"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	Client struct {
		ID   ClientID
		Name string

		CurrentVote *Vote
		HasVoted    bool
		VotedAt     *time.Time
		IsSpectator bool
		IsOwner     bool

		logger log.Logger
	}
)

func newClient(id ClientID) *Client {
	return &Client{
		ID:     id,
		logger: log.NewLogger("planningpoker.client"),
	}
}

func (c *Client) WithLogger(logger log.Logger) *Client {
	c.logger = logger
	return c
}

func (c *Client) applyVote(vote *Vote) {
	c.CurrentVote = vote
	if vote != nil && !vote.IsEmpty() {
		c.HasVoted = true
		now := time.Now().UTC()
		c.VotedAt = &now
	} else {
		c.HasVoted = false
		c.VotedAt = nil
	}
}

func (c *Client) UpdateName(ctx context.Context, name string) {
	c.Name = name
}
