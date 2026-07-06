package redis

import (
	"encoding/json"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	SerializedStory struct {
		Name               string   `json:"name"`
		Result             *float32 `json:"result,omitempty"`
		MostAppearingVotes []int    `json:"mostAppearingVotes"`
		Voted              bool     `json:"voted"`
	}
	SerializedRoom struct {
		ID                 string             `json:"id"`
		Clients            []SerializedClient `json:"clients"`
		CurrentStory       string             `json:"currentStory"`
		Reveal             bool               `json:"reveal"`
		Result             *float32           `json:"result,omitempty"`
		MostAppearingVotes []int              `json:"mostAppearingVotes"`
		BacklogMode        bool               `json:"backlogMode"`
		Stories            []SerializedStory  `json:"stories,omitempty"`
		CurrentStoryIndex  int                `json:"currentStoryIndex"`
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
		ID:                 room.ID,
		Clients:            clients,
		CurrentStory:       room.CurrentStory,
		Reveal:             room.Reveal,
		Result:             room.Result,
		MostAppearingVotes: room.MostAppearingVotes,
		BacklogMode:        room.BacklogMode,
		Stories:            serializeStories(room.Stories),
		CurrentStoryIndex:  room.CurrentStoryIndex,
	}

	return json.Marshal(serialized)
}

func serializeStories(stories []entity.Story) []SerializedStory {
	result := make([]SerializedStory, len(stories))
	for i, s := range stories {
		result[i] = SerializedStory{
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

	room := &entity.Room{
		ID:                 serialized.ID,
		Clients:            clientCollection,
		CurrentStory:       serialized.CurrentStory,
		Reveal:             serialized.Reveal,
		Result:             serialized.Result,
		MostAppearingVotes: serialized.MostAppearingVotes,
		BacklogMode:        serialized.BacklogMode,
		Stories:            deserializeStories(serialized.Stories),
		CurrentStoryIndex:  serialized.CurrentStoryIndex,
	}

	for _, sc := range serialized.Clients {
		client := sc.Client(room)
		room.Clients.Add(client)
	}

	return room, nil
}

func deserializeStories(stories []SerializedStory) []entity.Story {
	result := make([]entity.Story, len(stories))
	for i, s := range stories {
		result[i] = entity.Story{
			Name:               s.Name,
			Result:             s.Result,
			MostAppearingVotes: s.MostAppearingVotes,
			Voted:              s.Voted,
		}
	}
	return result
}
