package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"

	"planning-poker/internal/domain/entity"
)

type (
	SerializedStory struct {
		ID                 string   `json:"id,omitempty"`
		Name               string   `json:"name"`
		Result             *float32 `json:"result,omitempty"`
		MostAppearingVotes []string `json:"mostAppearingVotes"`
		Voted              bool     `json:"voted"`
	}
	SerializedRoom struct {
		ID                 string             `json:"id"`
		StartedAt          *time.Time         `json:"startedAt,omitempty"`
		Clients            []SerializedClient `json:"clients"`
		DeckType           entity.DeckType    `json:"deckType,omitempty"`
		CurrentStory       string             `json:"currentStory"`
		Reveal             bool               `json:"reveal"`
		Result             *float32           `json:"result,omitempty"`
		MostAppearingVotes []string           `json:"mostAppearingVotes"`
		Consensus          string             `json:"consensus,omitempty"`
		LowestVote         *string            `json:"lowestVote,omitempty"`
		HighestVote        *string            `json:"highestVote,omitempty"`
		VoteRange          *int               `json:"voteRange,omitempty"`
		VoteSpread         *int               `json:"voteSpread,omitempty"`
		SpecialVoteCount   int                `json:"specialVoteCount,omitempty"`
		BacklogMode        bool               `json:"backlogMode"`
		Stories            []SerializedStory  `json:"stories,omitempty"`
		CurrentStoryIndex  int                `json:"currentStoryIndex"`
		RoomVersion        uint64             `json:"roomVersion"`
	}
	SerializedClient struct {
		ID          string     `json:"id"`
		Name        string     `json:"name"`
		CurrentVote *string    `json:"currentVote,omitempty"`
		HasVoted    bool       `json:"hasVoted"`
		VotedAt     *time.Time `json:"votedAt,omitempty"`
		IsSpectator bool       `json:"isSpectator"`
		IsOwner     bool       `json:"isOwner"`
	}
)

func SerializeRoom(room *entity.Room) ([]byte, error) {
	clients := make([]SerializedClient, 0, room.Clients().Count())
	room.Clients().ForEach(func(client *entity.Client) {
		clients = append(clients, SerializedClient{
			ID:          client.ID,
			Name:        client.Name,
			CurrentVote: entity.OptionalVoteLabel(client.CurrentVote),
			HasVoted:    client.HasVoted,
			VotedAt:     entity.OptionalUTCTimePtr(client.VotedAt),
			IsSpectator: client.IsSpectator,
			IsOwner:     client.IsOwner,
		})
	})

	outcome := room.Outcome()
	serialized := SerializedRoom{
		ID:                 room.ID(),
		StartedAt:          entity.OptionalUTCTime(room.StartedAt()),
		Clients:            clients,
		DeckType:           room.DeckType,
		CurrentStory:       room.Snapshot().CurrentStory,
		Reveal:             room.Revealed(),
		Result:             outcomeResult(outcome),
		MostAppearingVotes: outcomeMostAppearingVotes(outcome),
		Consensus:          outcomeConsensus(outcome),
		LowestVote:         outcomeLowestVote(outcome),
		HighestVote:        outcomeHighestVote(outcome),
		VoteRange:          outcomeVoteRange(outcome),
		VoteSpread:         outcomeVoteSpread(outcome),
		SpecialVoteCount:   outcomeSpecialVoteCount(outcome),
		BacklogMode:        room.BacklogMode(),
		Stories:            serializeStories(room.Stories()),
		CurrentStoryIndex:  room.CurrentStoryIndex(),
		RoomVersion:        room.RoomVersion,
	}

	return json.Marshal(serialized)
}

func outcomeResult(outcome *entity.RoundOutcome) *float32 {
	if outcome == nil {
		return nil
	}
	return outcome.Result
}

func outcomeMostAppearingVotes(outcome *entity.RoundOutcome) []string {
	if outcome == nil {
		return nil
	}
	return outcome.MostAppearingVotes
}

func outcomeConsensus(outcome *entity.RoundOutcome) string {
	if outcome == nil {
		return ""
	}
	return outcome.Consensus
}

func outcomeLowestVote(outcome *entity.RoundOutcome) *string {
	if outcome == nil {
		return nil
	}
	return outcome.LowestVote
}

func outcomeHighestVote(outcome *entity.RoundOutcome) *string {
	if outcome == nil {
		return nil
	}
	return outcome.HighestVote
}

func outcomeVoteRange(outcome *entity.RoundOutcome) *int {
	if outcome == nil {
		return nil
	}
	return outcome.VoteRange
}

func outcomeVoteSpread(outcome *entity.RoundOutcome) *int {
	if outcome == nil {
		return nil
	}
	return outcome.VoteSpread
}

func outcomeSpecialVoteCount(outcome *entity.RoundOutcome) int {
	if outcome == nil {
		return 0
	}
	return outcome.SpecialVoteCount
}

func serializeStories(stories []entity.Story) []SerializedStory {
	result := make([]SerializedStory, len(stories))
	for i, s := range stories {
		result[i] = SerializedStory{
			ID:                 s.ID,
			Name:               s.Name,
			Result:             s.Result,
			MostAppearingVotes: s.MostAppearingVotes,
			Voted:              s.Voted,
		}
	}
	return result
}

func DeserializeRoom(data []byte, clientCollection entity.ClientCollection) (*entity.Room, error) {
	var serialized SerializedRoom
	if err := json.Unmarshal(data, &serialized); err != nil {
		return nil, err
	}
	stories, err := deserializeStories(serialized.Stories)
	if err != nil {
		return nil, err
	}

	room := entity.NewRoomFromSnapshot(entity.RoomSnapshot{
		ID:                serialized.ID,
		StartedAt:         entity.UTCTimeOrZero(serialized.StartedAt),
		DeckType:          serialized.DeckType,
		Version:           serialized.RoomVersion,
		CurrentStory:      serialized.CurrentStory,
		Revealed:          serialized.Reveal,
		Outcome:           outcomeFromSerialized(serialized),
		BacklogMode:       serialized.BacklogMode,
		Stories:           stories,
		CurrentStoryIndex: serialized.CurrentStoryIndex,
	}, clientCollection)

	for _, sc := range serialized.Clients {
		client := (&entity.Client{
			ID:          sc.ID,
			Name:        sc.Name,
			CurrentVote: entity.VotePtrFromLabel(sc.CurrentVote),
			HasVoted:    sc.HasVoted,
			VotedAt:     entity.OptionalUTCTimePtr(sc.VotedAt),
			IsSpectator: sc.IsSpectator,
			IsOwner:     sc.IsOwner,
		}).WithLogger(log.NewLogger("planningpoker.client"))
		room.Clients().Add(client)
	}
	return room, nil
}

// outcomeFromSerialized rebuilds the round outcome from the persisted fields.
// Hidden rounds have no outcome, so their metrics stay zero.
func outcomeFromSerialized(serialized SerializedRoom) *entity.RoundOutcome {
	if !serialized.Reveal {
		return nil
	}

	return &entity.RoundOutcome{
		Result:             serialized.Result,
		MostAppearingVotes: serialized.MostAppearingVotes,
		Consensus:          serialized.Consensus,
		LowestVote:         serialized.LowestVote,
		HighestVote:        serialized.HighestVote,
		VoteRange:          serialized.VoteRange,
		VoteSpread:         serialized.VoteSpread,
		SpecialVoteCount:   serialized.SpecialVoteCount,
	}
}

func deserializeStories(stories []SerializedStory) ([]entity.Story, error) {
	result := make([]entity.Story, len(stories))
	for i, s := range stories {
		if s.ID == "" {
			return nil, fmt.Errorf("story at index %d is missing an ID", i)
		}
		result[i] = entity.Story{
			ID:                 s.ID,
			Name:               s.Name,
			Result:             s.Result,
			MostAppearingVotes: s.MostAppearingVotes,
			Voted:              s.Voted,
		}
	}
	return result, nil
}
