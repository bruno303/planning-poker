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
		MostAppearingVotes []int    `json:"mostAppearingVotes"`
		Voted              bool     `json:"voted"`
	}

	RoomState struct {
		Type                string        `json:"type"`
		StartedAt           *time.Time    `json:"startedAt,omitempty"`
		CurrentStory        string        `json:"currentStory"`
		Reveal              bool          `json:"reveal"`
		Result              *float32      `json:"result,omitempty"`
		MostAppearingVotes  []int         `json:"mostAppearingVotes"`
		Consensus           string        `json:"consensus,omitempty"`
		LowestVote          *int          `json:"lowestVote,omitempty"`
		HighestVote         *int          `json:"highestVote,omitempty"`
		VoteRange           *int          `json:"voteRange,omitempty"`
		VoteSpread          *int          `json:"voteSpread,omitempty"`
		NonNumericVoteCount int           `json:"nonNumericVoteCount,omitempty"`
		Participants        []Participant `json:"participants"`
		BacklogMode         bool          `json:"backlogMode"`
		Stories             []Story       `json:"stories"`
		CurrentStoryIndex   int           `json:"currentStoryIndex"`
		RoomVersion         uint64        `json:"roomVersion"`
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
		Type:                "room-state",
		StartedAt:           optionalUTCTime(room.StartedAt),
		CurrentStory:        room.EffectiveCurrentStory(),
		Reveal:              room.Reveal,
		Participants:        MapToParticipants(room.Clients.Values()),
		Result:              room.Result,
		MostAppearingVotes:  room.MostAppearingVotes,
		Consensus:           room.Consensus,
		LowestVote:          room.LowestVote,
		HighestVote:         room.HighestVote,
		VoteRange:           room.VoteRange,
		VoteSpread:          room.VoteSpread,
		NonNumericVoteCount: room.NonNumericVoteCount,
		BacklogMode:         room.BacklogMode,
		Stories:             mapStories(room.Stories),
		CurrentStoryIndex:   room.CurrentStoryIndex,
		RoomVersion:         room.RoomVersion,
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
				VotedAt:     optionalUTCTimePtr(client.VotedAt),
				IsSpectator: client.IsSpectator,
				IsOwner:     client.IsOwner,
			}
		},
	)
}

func optionalUTCTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	utc := value.UTC()
	return &utc
}

func optionalUTCTimePtr(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	utc := value.UTC()
	return &utc
}
