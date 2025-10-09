package dto

import (
	"planning-poker/internal/domain/entity"
	"reflect"
	"testing"

	"github.com/samber/lo"
	"go.uber.org/mock/gomock"
)

func TestNewRoomStateCommand(t *testing.T) {
	vote := "5"
	clients := []*entity.Client{
		{ID: "1", Name: "Alice", CurrentVote: &vote, HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "2", Name: "Bob", CurrentVote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
	}

	ctrl := gomock.NewController(t)
	clientCollection := entity.NewMockClientCollection(ctrl)

	clientCollection.EXPECT().Values().Return(clients).AnyTimes()

	owner := clients[0]
	room := &entity.Room{
		ID:           "room1",
		Owner:        "Alice",
		OwnerClient:  owner,
		Clients:      clientCollection,
		CurrentStory: "Story 1",
		Reveal:       true,
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

	owner := clients[1]

	participants := MapToParticipants(clients, owner)

	expected := []Participant{
		{ID: "1", Name: "Alice", Vote: lo.ToPtr("5"), HasVoted: true, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", Vote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", Vote: lo.ToPtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
	}

	if !reflect.DeepEqual(participants, expected) {
		t.Errorf("Expected %v, got %v", expected, participants)
	}
}
