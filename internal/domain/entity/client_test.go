package entity

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
)

func assertNewClient(t *testing.T, client *Client, wantID string) {
	t.Helper()

	if client == nil {
		t.Fatal("newClient() returned nil")
	}
	if client.ID != cid(wantID) {
		t.Errorf("newClient() ID = %v, want %v", client.ID, wantID)
	}
	if client.Name != "" {
		t.Errorf("newClient() Name = %v, want empty string", client.Name)
	}
	if client.CurrentVote != nil {
		t.Errorf("newClient() CurrentVote = %v, want nil", client.CurrentVote)
	}
	if client.HasVoted {
		t.Errorf("newClient() HasVoted = %v, want false", client.HasVoted)
	}
	if client.VotedAt != nil {
		t.Errorf("newClient() VotedAt = %v, want nil", client.VotedAt)
	}
	if client.IsSpectator {
		t.Errorf("newClient() IsSpectator = %v, want false", client.IsSpectator)
	}
	if client.IsOwner {
		t.Errorf("newClient() IsOwner = %v, want false", client.IsOwner)
	}
}

func assertVoteResult(t *testing.T, client *Client, wantVote *Vote, wantHasVoted bool) {
	t.Helper()

	if wantVote == nil {
		if client.CurrentVote != nil {
			t.Errorf("Vote() CurrentVote = %v, want nil", *client.CurrentVote)
		}
	} else if client.CurrentVote == nil || *client.CurrentVote != *wantVote {
		t.Errorf("Vote() CurrentVote = %v, want %v", client.CurrentVote, *wantVote)
	}
	if client.HasVoted != wantHasVoted {
		t.Errorf("Vote() HasVoted = %v, want %v", client.HasVoted, wantHasVoted)
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
	}{
		{
			name:     "should create client with valid ID",
			clientID: "client-1",
		},
		{
			name:     "should create client with UUID",
			clientID: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "should create client with empty ID",
			clientID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertNewClient(t, newClient(cid(tt.clientID)), tt.clientID)
		})
	}
}

func TestClient_Vote(t *testing.T) {
	vote1 := "1"
	vote2 := "2"
	vote3 := "3"
	vote5 := "5"
	vote8 := "8"
	emptyVote := ""

	tests := []struct {
		name         string
		vote         *Vote
		wantVote     *Vote
		wantHasVoted bool
	}{
		{
			name:         "should accept vote 1",
			vote:         VotePtr(vote1),
			wantVote:     VotePtr(vote1),
			wantHasVoted: true,
		},
		{
			name:         "should accept vote 2",
			vote:         VotePtr(vote2),
			wantVote:     VotePtr(vote2),
			wantHasVoted: true,
		},
		{
			name:         "should accept vote 3",
			vote:         VotePtr(vote3),
			wantVote:     VotePtr(vote3),
			wantHasVoted: true,
		},
		{
			name:         "should accept vote 5",
			vote:         VotePtr(vote5),
			wantVote:     VotePtr(vote5),
			wantHasVoted: true,
		},
		{
			name:         "should accept vote 8",
			vote:         VotePtr(vote8),
			wantVote:     VotePtr(vote8),
			wantHasVoted: true,
		},
		{
			name:         "should accept nil vote",
			vote:         nil,
			wantVote:     nil,
			wantHasVoted: false,
		},
		{
			name:         "should accept empty vote",
			vote:         VotePtr(emptyVote),
			wantVote:     VotePtr(emptyVote),
			wantHasVoted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newClient("test-client")

			client.applyVote(tt.vote)

			assertVoteResult(t, client, tt.wantVote, tt.wantHasVoted)
		})
	}
}

func TestClient_Vote_UpdateVote(t *testing.T) {
	vote1 := "1"
	vote5 := "5"
	vote8 := "8"

	client := newClient("test-client")

	// First vote
	client.applyVote(VotePtr(vote1))
	if client.CurrentVote == nil || client.CurrentVote.String() != vote1 {
		t.Errorf("First Vote() CurrentVote = %v, want %v", client.CurrentVote, vote1)
	}
	if !client.HasVoted {
		t.Error("First Vote() HasVoted = false, want true")
	}

	// Change vote
	client.applyVote(VotePtr(vote5))
	if client.CurrentVote == nil || client.CurrentVote.String() != vote5 {
		t.Errorf("Second Vote() CurrentVote = %v, want %v", client.CurrentVote, vote5)
	}
	if !client.HasVoted {
		t.Error("Second Vote() HasVoted = false, want true")
	}

	// Change vote again
	client.applyVote(VotePtr(vote8))
	if client.CurrentVote == nil || client.CurrentVote.String() != vote8 {
		t.Errorf("Third Vote() CurrentVote = %v, want %v", client.CurrentVote, vote8)
	}
	if !client.HasVoted {
		t.Error("Third Vote() HasVoted = false, want true")
	}

	// Clear vote
	client.applyVote(nil)
	if client.CurrentVote != nil {
		t.Errorf("Clear Vote() CurrentVote = %v, want nil", client.CurrentVote)
	}
	if client.HasVoted {
		t.Error("Clear Vote() HasVoted = true, want false")
	}
}

func TestClient_UpdateName(t *testing.T) {
	tests := []struct {
		name        string
		initialName string
		newName     string
	}{
		{
			name:        "should update name from empty to John",
			initialName: "",
			newName:     "John",
		},
		{
			name:        "should update name from John to Jane",
			initialName: "John",
			newName:     "Jane",
		},
		{
			name:        "should update name to empty string",
			initialName: "John",
			newName:     "",
		},
		{
			name:        "should update name with special characters",
			initialName: "John",
			newName:     "José María",
		},
		{
			name:        "should update name with long string",
			initialName: "John",
			newName:     "A very long name that contains many characters and spaces",
		},
		{
			name:        "should update name with numbers",
			initialName: "John",
			newName:     "User123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := newClient("test-client")
			client.Name = tt.initialName

			client.UpdateName(ctx, tt.newName)

			if client.Name != tt.newName {
				t.Errorf("UpdateName() Name = %v, want %v", client.Name, tt.newName)
			}
		})
	}
}

func TestClient_UpdateName_Multiple(t *testing.T) {
	ctx := context.Background()
	client := newClient("test-client")

	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve"}

	for _, name := range names {
		client.UpdateName(ctx, name)
		if client.Name != name {
			t.Errorf("UpdateName() Name = %v, want %v", client.Name, name)
		}
	}
}

func TestClient_VoteBeforeAndAfterReveal(t *testing.T) {
	vote5 := "5"
	vote8 := "8"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := newClient("test-client")
	room := NewRoom(NewMockClientCollection(ctrl))

	client.applyVote(VotePtr(vote5))
	assertVoteResult(t, client, VotePtr(vote5), true)

	room.Reveal = true
	if room.applyVote(client, VotePtr(vote8)) {
		t.Error("applyVote() accepted a vote while revealed")
	}
	assertVoteResult(t, client, VotePtr(vote5), true)
}

func TestClient_SpectatorAndOwnerFlags(t *testing.T) {
	client := newClient("test-client")

	if client.IsSpectator {
		t.Error("Initial IsSpectator = true, want false")
	}
	if client.IsOwner {
		t.Error("Initial IsOwner = true, want false")
	}

	client.IsSpectator = true
	if !client.IsSpectator {
		t.Error("IsSpectator = false after setting, want true")
	}

	client.IsOwner = true
	if !client.IsOwner {
		t.Error("IsOwner = false after setting, want true")
	}
}
