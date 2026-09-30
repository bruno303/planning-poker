package dto

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"

	"planning-poker/internal/domain/entity"
)

type (
	Story struct {
		ID                 string   `json:"id"`
		Name               string   `json:"name"`
		Result             *float32 `json:"result,omitempty"`
		MostCommonVotes    []string `json:"mostCommonVotes"`
		MostAppearingVotes []int    `json:"mostAppearingVotes"`
		Voted              bool     `json:"voted"`
	}

	VotingDeck struct {
		ID    entity.DeckID   `json:"id"`
		Name  string          `json:"name"`
		Kind  entity.DeckKind `json:"kind"`
		Cards []string        `json:"cards"`
	}

	RoomState struct {
		Type                string        `json:"type"`
		Deck                VotingDeck    `json:"deck"`
		StartedAt           *time.Time    `json:"startedAt,omitempty"`
		CurrentStory        string        `json:"currentStory"`
		Reveal              bool          `json:"reveal"`
		Result              *float32      `json:"result,omitempty"`
		MostCommonVotes     []string      `json:"mostCommonVotes"`
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
	deck := room.Deck()
	return RoomState{
		Type:                "room-state",
		Deck:                VotingDeck{ID: deck.ID, Name: deck.Name, Kind: deck.Kind, Cards: deck.CardValues()},
		StartedAt:           entity.OptionalUTCTime(room.StartedAt()),
		CurrentStory:        room.EffectiveCurrentStory(),
		Reveal:              room.Reveal,
		Participants:        MapToParticipants(room.Clients.Values()),
		Result:              room.Result,
		MostCommonVotes:     nonNilVotes(room.MostCommonVotes),
		MostAppearingVotes:  legacyNumericModes(deck, room.MostCommonVotes),
		Consensus:           room.Consensus,
		LowestVote:          room.LowestVote,
		HighestVote:         room.HighestVote,
		VoteRange:           room.VoteRange,
		VoteSpread:          room.VoteSpread,
		NonNumericVoteCount: room.NonNumericVoteCount,
		BacklogMode:         room.BacklogMode,
		Stories:             mapStories(room.Stories, deck),
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

func mapStories(stories []entity.Story, deck entity.Deck) []Story {
	return lo.Map(stories, func(s entity.Story, _ int) Story {
		return Story{
			ID:                 s.ID,
			Name:               s.Name,
			Result:             s.Result,
			MostCommonVotes:    nonNilVotes(s.MostCommonVotes),
			MostAppearingVotes: legacyNumericModes(deck, s.MostCommonVotes),
			Voted:              s.Voted,
		}
	})
}

func nonNilVotes(votes []string) []string {
	return append([]string{}, votes...)
}

func legacyNumericModes(deck entity.Deck, votes []string) []int {
	if deck.Kind != entity.DeckKindNumeric {
		return []int{}
	}
	var result []int
	for _, vote := range votes {
		if numericVote, err := strconv.Atoi(vote); err == nil {
			result = append(result, numericVote)
		}
	}
	return result
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
