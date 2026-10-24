package entity

import (
	"context"
	"testing"
)

func assertNewClient(t *testing.T, client *Client, wantID string) {
	t.Helper()

	if client == nil {
		t.Fatal("newClient() returned nil")
	}
	if client.ID != wantID {
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
	if client.logger == nil {
		t.Error("newClient() logger is nil")
	}
}

func assertVoteResult(t *testing.T, client *Client, wantVote *Vote, wantHasVoted bool) {
	t.Helper()

	if wantVote == nil {
		if client.CurrentVote != nil {
			t.Errorf("Vote() CurrentVote = %v, want nil", client.CurrentVote.Label())
		}
	} else if client.CurrentVote == nil || client.CurrentVote.Label() != wantVote.Label() {
		t.Errorf("Vote() CurrentVote = %v, want %v", client.CurrentVote, wantVote.Label())
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
			assertNewClient(t, newClient(tt.clientID), tt.clientID)
		})
	}
}

func TestClient_Vote(t *testing.T) {
	vote1 := VotePtr("1")
	vote2 := VotePtr("2")
	vote3 := VotePtr("3")
	vote5 := VotePtr("5")
	vote8 := VotePtr("8")
	emptyVote := VotePtr("")

	tests := []struct {
		name         string
		reveal       bool
		vote         *Vote
		wantVote     *Vote
		wantHasVoted bool
	}{
		{
			name:         "should accept vote when not revealed - vote 1",
			reveal:       false,
			vote:         vote1,
			wantVote:     vote1,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 2",
			reveal:       false,
			vote:         vote2,
			wantVote:     vote2,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 3",
			reveal:       false,
			vote:         vote3,
			wantVote:     vote3,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 5",
			reveal:       false,
			vote:         vote5,
			wantVote:     vote5,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 8",
			reveal:       false,
			vote:         vote8,
			wantVote:     vote8,
			wantHasVoted: true,
		},
		{
			name:         "should accept nil vote when not revealed",
			reveal:       false,
			vote:         nil,
			wantVote:     nil,
			wantHasVoted: false,
		},
		{
			name:         "should accept empty vote when not revealed",
			reveal:       false,
			vote:         emptyVote,
			wantVote:     emptyVote,
			wantHasVoted: false,
		},
		{
			name:         "should ignore vote when revealed",
			reveal:       true,
			vote:         vote5,
			wantVote:     nil,
			wantHasVoted: false,
		},
		{
			name:         "should ignore nil vote when revealed",
			reveal:       true,
			vote:         nil,
			wantVote:     nil,
			wantHasVoted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := newClient("test-client")

			room := NewRoom(newTestCollection(client))
			room.reveal(tt.reveal)

			if err := room.Vote(ctx, client.ID, tt.vote); err != nil {
				t.Fatalf("Vote() error = %v", err)
			}

			assertVoteResult(t, client, tt.wantVote, tt.wantHasVoted)
		})
	}
}

func TestClient_Vote_UpdateVote(t *testing.T) {
	ctx := context.Background()

	vote1 := VotePtr("1")
	vote5 := VotePtr("5")
	vote8 := VotePtr("8")

	client := newClient("test-client")
	other := newClient("other-client")
	room := NewRoom(newTestCollection(client, other))
	room.reveal(false)

	// First vote
	if err := room.Vote(ctx, client.ID, vote1); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
	if client.CurrentVote == nil || client.CurrentVote.Label() != vote1.Label() {
		t.Errorf("First Vote() CurrentVote = %v, want %v", client.CurrentVote, vote1.Label())
	}
	if !client.HasVoted {
		t.Error("First Vote() HasVoted = false, want true")
	}

	// Change vote
	if err := room.Vote(ctx, client.ID, vote5); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
	if client.CurrentVote == nil || client.CurrentVote.Label() != vote5.Label() {
		t.Errorf("Second Vote() CurrentVote = %v, want %v", client.CurrentVote, vote5.Label())
	}
	if !client.HasVoted {
		t.Error("Second Vote() HasVoted = false, want true")
	}

	// Change vote again
	if err := room.Vote(ctx, client.ID, vote8); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
	if client.CurrentVote == nil || client.CurrentVote.Label() != vote8.Label() {
		t.Errorf("Third Vote() CurrentVote = %v, want %v", client.CurrentVote, vote8.Label())
	}
	if !client.HasVoted {
		t.Error("Third Vote() HasVoted = false, want true")
	}

	// Clear vote
	if err := room.Vote(ctx, client.ID, nil); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
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
	ctx := context.Background()
	vote5 := VotePtr("5")
	vote8 := VotePtr("8")

	client := newClient("test-client")
	room := NewRoom(newTestCollection(client))
	room.reveal(false)

	if err := room.Vote(ctx, client.ID, vote5); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
	assertVoteResult(t, client, vote5, true)

	room.reveal(true)
	if err := room.Vote(ctx, client.ID, vote8); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
	assertVoteResult(t, client, vote5, true)
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
