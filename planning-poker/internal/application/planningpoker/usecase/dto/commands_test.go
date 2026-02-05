package dto

import (
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/clientcollection"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestNewRoomStateCommand(t *testing.T) {
	vote := "5"
	clients := []*entity.Client{
		{ID: "1", Name: "Alice", CurrentVote: &vote, HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "2", Name: "Bob", CurrentVote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
	}

	clientCollection := clientcollection.New()
	for _, client := range clients {
		clientCollection.Add(client)
	}

	room := &entity.Room{
		ID:                 "room1",
		Clients:            clientCollection,
		CurrentStory:       "Story 1",
		Reveal:             true,
		Result:             lo.ToPtr(float32(5)),
		MostAppearingVotes: []int{1, 2},
	}
	got := NewRoomStateCommand(room)
	want := RoomState{
		Type:         "room-state",
		CurrentStory: "Story 1",
		Reveal:       true,
		Participants: []Participant{
			{ID: "1", Name: "Alice", Vote: &vote, HasVoted: true, IsSpectator: false, IsOwner: true},
			{ID: "2", Name: "Bob", Vote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
		},
		Result:             lo.ToPtr(float32(5)),
		MostAppearingVotes: []int{1, 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewRoomStateCommand() = %+v, want %+v", got, want)
	}
}

func TestNewUpdateClientIDCommand(t *testing.T) {
	got := NewUpdateClientIDCommand("client-123")
	want := UpdateClientID{
		Type:     "update-client-id",
		ClientID: "client-123",
	}
	if got != want {
		t.Errorf("NewUpdateClientIDCommand() = %+v, want %+v", got, want)
	}
}

func TestMapToParticipants(t *testing.T) {
	clients := []*entity.Client{
		{ID: "1", Name: "Alice", CurrentVote: lo.ToPtr("5"), HasVoted: true, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", CurrentVote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", CurrentVote: lo.ToPtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
	}
	participants := MapToParticipants(clients)

	expected := []Participant{
		{ID: "1", Name: "Alice", Vote: lo.ToPtr("5"), HasVoted: true, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", Vote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", Vote: lo.ToPtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
	}

	if !reflect.DeepEqual(participants, expected) {
		t.Errorf("Expected %v, got %v", expected, participants)
	}
}
