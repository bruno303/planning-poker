package planningpoker

import (
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestMapToParticipants(t *testing.T) {
	clients := map[string]*Client{
		"1": {ID: "1", Name: "Alice", Vote: lo.ToPtr("5"), HasVoted: true, IsSpectator: false, IsOwner: false},
		"2": {ID: "2", Name: "Bob", Vote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		"3": {ID: "3", Name: "Charlie", Vote: lo.ToPtr("?"), HasVoted: false, IsSpectator: true, IsOwner: false},
	}
	owner := clients["2"]

	participants := MapToParticipants(clients, owner)

	expected := []Participant{
		{ID: "1", Name: "Alice", Vote: lo.ToPtr("5"), HasVoted: true, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", Vote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", Vote: lo.ToPtr("?"), HasVoted: false, IsSpectator: true, IsOwner: false},
	}

	if !reflect.DeepEqual(participants, expected) {
		t.Errorf("Expected %v, got %v", expected, participants)
	}
}
