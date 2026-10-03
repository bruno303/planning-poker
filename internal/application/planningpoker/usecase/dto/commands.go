package dto

import (
	"time"

	"planning-poker/internal/domain/entity"
	"slices"
	"strings"

	"github.com/samber/lo"
)

type (
	Story struct {
		ID                 string   `json:"id"`
		Name               string   `json:"name"`
		Result             *float32 `json:"result,omitempty"`
		MostAppearingVotes []string `json:"mostAppearingVotes"`
		Voted              bool     `json:"voted"`
	}

	RoomState struct {
		Type               string        `json:"type"`
		StartedAt          *time.Time    `json:"startedAt,omitempty"`
		Deck               entity.Deck   `json:"deck"`
		CurrentStory       string        `json:"currentStory"`
		Reveal             bool          `json:"reveal"`
		Result             *float32      `json:"result,omitempty"`
		MostAppearingVotes []string      `json:"mostAppearingVotes"`
		Consensus          string        `json:"consensus,omitempty"`
		LowestVote         *string       `json:"lowestVote,omitempty"`
		HighestVote        *string       `json:"highestVote,omitempty"`
		VoteRange          *int          `json:"voteRange,omitempty"`
		VoteSpread         *int          `json:"voteSpread,omitempty"`
		SpecialVoteCount   int           `json:"specialVoteCount,omitempty"`
		Participants       []Participant `json:"participants"`
		BacklogMode        bool          `json:"backlogMode"`
		Stories            []Story       `json:"stories"`
		CurrentStoryIndex  int           `json:"currentStoryIndex"`
		RoomVersion        uint64        `json:"roomVersion"`
	}
	Participant struct {
		ID          string     `json:"id"`
		Name        string     `json:"name"`
		Vote        *string    `json:"vote"`
		HasVoted    bool       `json:"hasVoted"`
		VotedAt     *time.Time `json:"votedAt,omitempty"`
		IsSpectator bool       `json:"isSpectator"`
		IsOwner     bool       `json:"isOwner"`
	}

	UpdateClientID struct {
		Type     string `json:"type"`
		ClientID string `json:"clientId"`
	}

	KickNotification struct {
		Type string `json:"type"`
	}
)

func NewRoomStateCommand(room *entity.Room) RoomState {
	return RoomState{
		Type:               "room-state",
		StartedAt:          entity.OptionalUTCTime(room.StartedAt()),
		Deck:               room.Deck(),
		CurrentStory:       room.EffectiveCurrentStory(),
		Reveal:             room.Reveal,
		Participants:       MapToParticipants(room.Clients.Values()),
		Result:             room.Result,
		MostAppearingVotes: room.MostAppearingVotes,
		Consensus:          room.Consensus,
		LowestVote:         room.LowestVote,
		HighestVote:        room.HighestVote,
		VoteRange:          room.VoteRange,
		VoteSpread:         room.VoteSpread,
		SpecialVoteCount:   room.SpecialVoteCount,
		BacklogMode:        room.BacklogMode,
		Stories:            mapStories(room.Stories),
		CurrentStoryIndex:  room.CurrentStoryIndex,
		RoomVersion:        room.RoomVersion,
	}
}

func NewUpdateClientIDCommand(clientID string) UpdateClientID {
	return UpdateClientID{
		Type:     "update-client-id",
		ClientID: clientID,
	}
}

func NewKickNotification() KickNotification {
	return KickNotification{
		Type: "kicked",
	}
}

func mapStories(stories []entity.Story) []Story {
	return lo.Map(stories, func(s entity.Story, _ int) Story {
		return Story{
			ID:                 s.ID,
			Name:               s.Name,
			Result:             s.Result,
			MostAppearingVotes: s.MostAppearingVotes,
			Voted:              s.Voted,
		}
	})
}

func MapToParticipants(clients []*entity.Client) []Participant {
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
				VotedAt:     entity.OptionalUTCTimePtr(client.VotedAt),
				IsSpectator: client.IsSpectator,
				IsOwner:     client.IsOwner,
			}
		},
	)
}
