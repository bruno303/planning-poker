package dto

import (
	"planning-poker/internal/domain/entity"
	"slices"
	"strings"

	"github.com/samber/lo"
)

type (
	RoomState struct {
		Type         string        `json:"type"`
		CurrentStory string        `json:"currentStory"`
		Reveal       bool          `json:"reveal"`
		Participants []Participant `json:"participants"`
	}
	Participant struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Vote        *string `json:"vote"`
		HasVoted    bool    `json:"hasVoted"`
		IsSpectator bool    `json:"isSpectator"`
		IsOwner     bool    `json:"isOwner"`
	}

	UpdateClientID struct {
		Type     string `json:"type"`
		ClientID string `json:"clientId"`
	}
)

func NewRoomStateCommand(room *entity.Room) RoomState {
	return RoomState{
		Type:         "room-state",
		CurrentStory: room.CurrentStory,
		Reveal:       room.Reveal,
		Participants: MapToParticipants(room.Clients.Values(), room.OwnerClient),
	}
}

func NewUpdateClientIDCommand(clientID string) UpdateClientID {
	return UpdateClientID{
		Type:     "update-client-id",
		ClientID: clientID,
	}
}

func MapToParticipants(clients []*entity.Client, owner *entity.Client) []Participant {
	slices.SortFunc(clients, func(a, b *entity.Client) int {
		return strings.Compare(a.Name, b.Name)
	})

	return lo.Map(
		clients,
		func(client *entity.Client, _ int) Participant {
			return Participant{
				ID:          client.ID,
				Name:        client.Name,
				Vote:        client.CurrentVote,
				HasVoted:    client.HasVoted,
				IsSpectator: client.IsSpectator,
				IsOwner:     client.IsOwner,
			}
		},
	)
}
