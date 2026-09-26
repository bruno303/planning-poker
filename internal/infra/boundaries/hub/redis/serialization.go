package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"

	"planning-poker/internal/domain/entity"
)

type (
	// SerializedVotes tolerates legacy records that stored most appearing
	// votes as JSON numbers while always marshaling them back as strings.
	SerializedVotes []string

	SerializedStory struct {
		ID                 string          `json:"id,omitempty"`
		Name               string          `json:"name"`
		Result             *float64        `json:"result,omitempty"`
		MostAppearingVotes SerializedVotes `json:"mostAppearingVotes"`
		Voted              bool            `json:"voted"`
	}
	SerializedRoom struct {
		ID                  string             `json:"id"`
		Deck                []string           `json:"deck,omitempty"`
		DeckPreset          string             `json:"deckPreset,omitempty"`
		StartedAt           *time.Time         `json:"startedAt,omitempty"`
		Clients             []SerializedClient `json:"clients"`
		CurrentStory        string             `json:"currentStory"`
		Reveal              bool               `json:"reveal"`
		Result              *float64           `json:"result,omitempty"`
		MostAppearingVotes  SerializedVotes    `json:"mostAppearingVotes"`
		Consensus           string             `json:"consensus,omitempty"`
		LowestVote          *float64           `json:"lowestVote,omitempty"`
		HighestVote         *float64           `json:"highestVote,omitempty"`
		VoteRange           *float64           `json:"voteRange,omitempty"`
		VoteSpread          *int               `json:"voteSpread,omitempty"`
		NonNumericVoteCount int                `json:"nonNumericVoteCount,omitempty"`
		BacklogMode         bool               `json:"backlogMode"`
		Stories             []SerializedStory  `json:"stories,omitempty"`
		CurrentStoryIndex   int                `json:"currentStoryIndex"`
		RoomVersion         uint64             `json:"roomVersion"`
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

func (v *SerializedVotes) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw == nil {
		*v = nil
		return nil
	}

	votes := make(SerializedVotes, len(raw))
	for i, item := range raw {
		var label string
		if err := json.Unmarshal(item, &label); err == nil {
			votes[i] = label
			continue
		}

		var number json.Number
		if err := json.Unmarshal(item, &number); err != nil {
			return fmt.Errorf("invalid most appearing vote %s: %w", string(item), err)
		}
		votes[i] = number.String()
	}

	*v = votes
	return nil
}

func (sc SerializedClient) Client(room *entity.Room) *entity.Client {
	client := &entity.Client{
		ID:          sc.ID,
		Name:        sc.Name,
		CurrentVote: sc.CurrentVote,
		HasVoted:    sc.HasVoted,
		VotedAt:     entity.OptionalUTCTimePtr(sc.VotedAt),
		IsSpectator: sc.IsSpectator,
		IsOwner:     sc.IsOwner,
	}

	return client.
		WithRoom(room).
		WithLogger(log.NewLogger("planningpoker.client"))
}

func SerializeRoom(room *entity.Room) ([]byte, error) {
	clients := make([]SerializedClient, 0, room.Clients.Count())
	room.Clients.ForEach(func(client *entity.Client) {
		clients = append(clients, SerializedClient{
			ID:          client.ID,
			Name:        client.Name,
			CurrentVote: client.CurrentVote,
			HasVoted:    client.HasVoted,
			VotedAt:     entity.OptionalUTCTimePtr(client.VotedAt),
			IsSpectator: client.IsSpectator,
			IsOwner:     client.IsOwner,
		})
	})

	serialized := SerializedRoom{
		ID:                  room.ID,
		Deck:                room.Deck.Cards,
		DeckPreset:          room.Deck.ID,
		StartedAt:           entity.OptionalUTCTime(room.StartedAt()),
		Clients:             clients,
		CurrentStory:        room.CurrentStory,
		Reveal:              room.Reveal,
		Result:              room.Round.Average,
		MostAppearingVotes:  SerializedVotes(room.Round.MostCommon),
		Consensus:           room.Round.Consensus,
		LowestVote:          room.Round.Lowest,
		HighestVote:         room.Round.Highest,
		VoteRange:           room.Round.Range,
		VoteSpread:          room.Round.Spread,
		NonNumericVoteCount: room.Round.NonNumericCount,
		BacklogMode:         room.BacklogMode,
		Stories:             serializeStories(room.Stories),
		CurrentStoryIndex:   room.CurrentStoryIndex,
		RoomVersion:         room.RoomVersion,
	}

	return json.Marshal(serialized)
}

func serializeStories(stories []entity.Story) []SerializedStory {
	result := make([]SerializedStory, len(stories))
	for i, s := range stories {
		result[i] = SerializedStory{
			ID:                 s.ID,
			Name:               s.Name,
			Result:             s.Result,
			MostAppearingVotes: SerializedVotes(s.MostAppearingVotes),
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

	deck := entity.DefaultDeck()
	if len(serialized.Deck) > 0 {
		deck = entity.Deck{ID: serialized.DeckPreset, Cards: append([]string(nil), serialized.Deck...)}
		if preset, ok := entity.DeckByID(serialized.DeckPreset); ok {
			deck.Name = preset.Name
		}
	}
	room := entity.NewRoomWithIDAndStartedAt(
		serialized.ID,
		clientCollection,
		entity.UTCTimeOrZero(serialized.StartedAt),
	)
	room.Deck = deck
	room.CurrentStory = serialized.CurrentStory
	room.Reveal = serialized.Reveal
	room.Round = entity.RoundResult{
		Average:         serialized.Result,
		MostCommon:      []string(serialized.MostAppearingVotes),
		Consensus:       serialized.Consensus,
		Lowest:          serialized.LowestVote,
		Highest:         serialized.HighestVote,
		Range:           serialized.VoteRange,
		Spread:          serialized.VoteSpread,
		NonNumericCount: serialized.NonNumericVoteCount,
	}
	room.BacklogMode = serialized.BacklogMode
	room.Stories = stories
	room.CurrentStoryIndex = serialized.CurrentStoryIndex
	room.RoomVersion = serialized.RoomVersion

	for _, sc := range serialized.Clients {
		client := sc.Client(room)
		room.Clients.Add(client)
	}
	return room, nil
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
			MostAppearingVotes: []string(s.MostAppearingVotes),
			Voted:              s.Voted,
		}
	}
	return result, nil
}
