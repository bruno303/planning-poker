package entity

//go:generate go tool mockgen -destination mocks.go -package entity . ClientCollection

import (
	"time"

	"github.com/google/uuid"
)

type (
	Room struct {
		ID                 string
		startedAt          time.Time
		Clients            ClientCollection
		DeckType           DeckType
		CurrentStory       string
		Reveal             bool
		Result             *float32
		MostAppearingVotes []string
		Consensus          string
		LowestVote         *string
		HighestVote        *string
		VoteRange          *int
		VoteSpread         *int
		SpecialVoteCount   int
		BacklogMode        bool
		Stories            []Story
		CurrentStoryIndex  int
		RoomVersion        RoomVersion
	}
)

func NewRoom(clients ClientCollection) *Room {
	return NewRoomWithID(uuid.NewString(), clients)
}

func NewRoomWithID(id string, clients ClientCollection) *Room {
	return NewRoomWithIDAndStartedAt(id, clients, time.Now().UTC())
}

// NewRoomWithIDAndStartedAt creates a room with a persisted start time.
func NewRoomWithIDAndStartedAt(id string, clients ClientCollection, startedAt time.Time) *Room {
	return &Room{
		ID:           id,
		startedAt:    startedAt.UTC(),
		Clients:      clients,
		DeckType:     DeckTypeFibonacci,
		CurrentStory: "",
		Reveal:       false,
		Result:       nil,
		BacklogMode:  true,
	}
}

// NewRoomWithDeck creates a room fixed to the given voting deck.
func NewRoomWithDeck(clients ClientCollection, deckType DeckType) *Room {
	return NewRoomWithIDAndDeck(uuid.NewString(), clients, deckType)
}

// NewRoomWithIDAndDeck creates a room with an explicit ID fixed to the given
// voting deck.
func NewRoomWithIDAndDeck(id string, clients ClientCollection, deckType DeckType) *Room {
	room := NewRoomWithIDAndStartedAt(id, clients, time.Now().UTC())
	room.DeckType = deckType
	return room
}

// Deck returns the room's immutable deck descriptor. Unknown or empty deck
// types resolve to Fibonacci so repository tests and zero-value rooms stay
// usable.
func (r *Room) Deck() Deck {
	return deckForType(r.DeckType)
}

// StartedAt returns the room start time as a value copy.
func (r *Room) StartedAt() time.Time {
	return r.startedAt
}
