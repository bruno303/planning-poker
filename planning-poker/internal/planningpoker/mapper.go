package planningpoker

import (
	"slices"
	"strings"

	"github.com/samber/lo"
)

func MapToParticipants(clients map[string]*Client, owner *Client) []Participant {

	clis := lo.Values(clients)
	slices.SortFunc(clis, func(a, b *Client) int {
		return strings.Compare(a.Name, b.Name)
	})

	return lo.Map(
		clis,
		func(client *Client, _ int) Participant {
			return Participant{
				ID:          client.ID,
				Name:        client.Name,
				Vote:        client.Vote,
				HasVoted:    client.HasVoted,
				IsSpectator: client.IsSpectator,
				IsOwner:     client.IsOwner,
			}
		},
	)
}
