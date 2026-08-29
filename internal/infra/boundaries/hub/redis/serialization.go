package redis

import (
	"encoding/json"
	"fmt"

	"github.com/bruno303/go-toolkit/pkg/log"

	"planning-poker/internal/domain/entity"
)

type (
	SerializedStory struct {
		ID                 string   `json:"id,omitempty"`
		Name               string   `json:"name"`
		Result             *float32 `json:"result,omitempty"`
		MostAppearingVotes []int    `json:"mostAppearingVotes"`
		Voted              bool     `json:"voted"`
	}
	SerializedRoom struct {
		ID                  string             `json:"id"`
		Clients             []SerializedClient `json:"clients"`
		CurrentStory        string             `json:"currentStory"`
		Reveal              bool               `json:"reveal"`
		Result              *float32           `json:"result,omitempty"`
		MostAppearingVotes  []int              `json:"mostAppearingVotes"`
		Consensus           string             `json:"consensus,omitempty"`
		LowestVote          *int               `json:"lowestVote,omitempty"`
		HighestVote         *int               `json:"highestVote,omitempty"`
		VoteRange           *int               `json:"voteRange,omitempty"`
		VoteSpread          *int               `json:"voteSpread,omitempty"`
		NonNumericVoteCount int                `json:"nonNumericVoteCount,omitempty"`
		BacklogMode         bool               `json:"backlogMode"`
		Stories             []SerializedStory  `json:"stories,omitempty"`
		CurrentStoryIndex   int                `json:"currentStoryIndex"`
		RoomVersion         uint64             `json:"roomVersion"`
	}
	SerializedClient struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		CurrentVote *string `json:"currentVote,omitempty"`
		HasVoted    bool    `json:"hasVoted"`
		IsSpectator bool    `json:"isSpectator"`
		IsOwner     bool    `json:"isOwner"`
	}
)

func (sc SerializedClient) Client(room *entity.Room) *entity.Client {
	client := &entity.Client{
		ID:          sc.ID,
		Name:        sc.Name,
		CurrentVote: sc.CurrentVote,
		HasVoted:    sc.HasVoted,
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
			IsSpectator: client.IsSpectator,
			IsOwner:     client.IsOwner,
		})
	})

	serialized := SerializedRoom{
		ID:                  room.ID,
		Clients:             clients,
		CurrentStory:        room.CurrentStory,
		Reveal:              room.Reveal,
		Result:              room.Result,
		MostAppearingVotes:  room.MostAppearingVotes,
		Consensus:           room.Consensus,
		LowestVote:          room.LowestVote,
		HighestVote:         room.HighestVote,
		VoteRange:           room.VoteRange,
		VoteSpread:          room.VoteSpread,
		NonNumericVoteCount: room.NonNumericVoteCount,
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

	room := &entity.Room{
		ID:                  serialized.ID,
		Clients:             clientCollection,
		CurrentStory:        serialized.CurrentStory,
		Reveal:              serialized.Reveal,
		Result:              serialized.Result,
		MostAppearingVotes:  serialized.MostAppearingVotes,
		Consensus:           serialized.Consensus,
		LowestVote:          serialized.LowestVote,
		HighestVote:         serialized.HighestVote,
		VoteRange:           serialized.VoteRange,
		VoteSpread:          serialized.VoteSpread,
		NonNumericVoteCount: serialized.NonNumericVoteCount,
		BacklogMode:         serialized.BacklogMode,
		Stories:             stories,
		CurrentStoryIndex:   serialized.CurrentStoryIndex,
		RoomVersion:         serialized.RoomVersion,
	}

	for _, sc := range serialized.Clients {
		client := sc.Client(room)
		room.Clients.Add(client)
	}
	room.SetPersistedRoomVersion(room.RoomVersion)

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
			MostAppearingVotes: s.MostAppearingVotes,
			Voted:              s.Voted,
		}
	}
	return result, nil
}
