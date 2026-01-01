package entity

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
)

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
			client := newClient(tt.clientID)

			if client == nil {
				t.Fatal("newClient() returned nil")
			}

			if client.ID != tt.clientID {
				t.Errorf("newClient() ID = %v, want %v", client.ID, tt.clientID)
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

			if client.IsSpectator {
				t.Errorf("newClient() IsSpectator = %v, want false", client.IsSpectator)
			}

			if client.IsOwner {
				t.Errorf("newClient() IsOwner = %v, want false", client.IsOwner)
			}

			if client.logger == nil {
				t.Error("newClient() logger is nil")
			}
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
		reveal       bool
		vote         *string
		wantVote     *string
		wantHasVoted bool
	}{
		{
			name:         "should accept vote when not revealed - vote 1",
			reveal:       false,
			vote:         &vote1,
			wantVote:     &vote1,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 2",
			reveal:       false,
			vote:         &vote2,
			wantVote:     &vote2,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 3",
			reveal:       false,
			vote:         &vote3,
			wantVote:     &vote3,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 5",
			reveal:       false,
			vote:         &vote5,
			wantVote:     &vote5,
			wantHasVoted: true,
		},
		{
			name:         "should accept vote when not revealed - vote 8",
			reveal:       false,
			vote:         &vote8,
			wantVote:     &vote8,
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
			vote:         &emptyVote,
			wantVote:     &emptyVote,
			wantHasVoted: false,
		},
		{
			name:         "should ignore vote when revealed",
			reveal:       true,
			vote:         &vote5,
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

			// Create a room and attach to client
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockClientCollection := NewMockClientCollection(ctrl)
			room := NewRoom(mockClientCollection)
			room.Reveal = tt.reveal
			client.room = room

			client.Vote(ctx, tt.vote)

			if tt.reveal {
				// When reveal is true, vote should be ignored
				if client.CurrentVote != nil {
					t.Errorf("Vote() with reveal=true, CurrentVote = %v, want nil", client.CurrentVote)
				}
				if client.HasVoted {
					t.Errorf("Vote() with reveal=true, HasVoted = %v, want false", client.HasVoted)
				}
			} else {
				// When reveal is false, vote should be accepted
				if tt.wantVote == nil {
					if client.CurrentVote != nil {
						t.Errorf("Vote() CurrentVote = %v, want nil", *client.CurrentVote)
					}
				} else {
					if client.CurrentVote == nil {
						t.Errorf("Vote() CurrentVote = nil, want %v", *tt.wantVote)
					} else if *client.CurrentVote != *tt.wantVote {
						t.Errorf("Vote() CurrentVote = %v, want %v", *client.CurrentVote, *tt.wantVote)
					}
				}

				if client.HasVoted != tt.wantHasVoted {
					t.Errorf("Vote() HasVoted = %v, want %v", client.HasVoted, tt.wantHasVoted)
				}
			}
		})
	}
}

func TestClient_Vote_UpdateVote(t *testing.T) {
	ctx := context.Background()

	vote1 := "1"
	vote5 := "5"
	vote8 := "8"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := newClient("test-client")
	mockClientCollection := NewMockClientCollection(ctrl)
	room := NewRoom(mockClientCollection)
	room.Reveal = false
	client.room = room

	// First vote
	client.Vote(ctx, &vote1)
	if client.CurrentVote == nil || *client.CurrentVote != vote1 {
		t.Errorf("First Vote() CurrentVote = %v, want %v", client.CurrentVote, vote1)
	}
	if !client.HasVoted {
		t.Error("First Vote() HasVoted = false, want true")
	}

	// Change vote
	client.Vote(ctx, &vote5)
	if client.CurrentVote == nil || *client.CurrentVote != vote5 {
		t.Errorf("Second Vote() CurrentVote = %v, want %v", client.CurrentVote, vote5)
	}
	if !client.HasVoted {
		t.Error("Second Vote() HasVoted = false, want true")
	}

	// Change vote again
	client.Vote(ctx, &vote8)
	if client.CurrentVote == nil || *client.CurrentVote != vote8 {
		t.Errorf("Third Vote() CurrentVote = %v, want %v", client.CurrentVote, vote8)
	}
	if !client.HasVoted {
		t.Error("Third Vote() HasVoted = false, want true")
	}

	// Clear vote
	client.Vote(ctx, nil)
	if client.CurrentVote != nil {
		t.Errorf("Clear Vote() CurrentVote = %v, want nil", client.CurrentVote)
	}
	if client.HasVoted {
		t.Error("Clear Vote() HasVoted = true, want false")
	}
}

func TestClient_Room(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := newClient("test-client")
	mockClientCollection := NewMockClientCollection(ctrl)
	room := NewRoom(mockClientCollection)
	client.room = room

	gotRoom := client.Room()

	if gotRoom == nil {
		t.Fatal("Room() returned nil")
	}

	if gotRoom != room {
		t.Error("Room() returned different room instance")
	}

	if gotRoom.ID != room.ID {
		t.Errorf("Room() ID = %v, want %v", gotRoom.ID, room.ID)
	}
}

func TestClient_Room_Nil(t *testing.T) {
	client := newClient("test-client")

	gotRoom := client.Room()

	if gotRoom != nil {
		t.Errorf("Room() = %v, want nil", gotRoom)
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

func TestClient_VoteScenarios(t *testing.T) {
	t.Run("should handle voting before and after reveal", func(t *testing.T) {
		ctx := context.Background()
		vote5 := "5"
		vote8 := "8"

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := newClient("test-client")
		mockClientCollection := NewMockClientCollection(ctrl)
		room := NewRoom(mockClientCollection)
		room.Reveal = false
		client.room = room

		// Vote before reveal
		client.Vote(ctx, &vote5)
		if client.CurrentVote == nil || *client.CurrentVote != vote5 {
			t.Errorf("Vote before reveal: CurrentVote = %v, want %v", client.CurrentVote, vote5)
		}
		if !client.HasVoted {
			t.Error("Vote before reveal: HasVoted = false, want true")
		}

		// Reveal votes
		room.Reveal = true

		// Try to vote after reveal - should be ignored
		client.Vote(ctx, &vote8)
		if client.CurrentVote == nil || *client.CurrentVote != vote5 {
			t.Errorf("Vote after reveal: CurrentVote = %v, want %v (unchanged)", client.CurrentVote, vote5)
		}
		if !client.HasVoted {
			t.Error("Vote after reveal: HasVoted = false, want true (unchanged)")
		}
	})

	t.Run("should handle spectator and owner flags", func(t *testing.T) {
		client := newClient("test-client")

		// Initially not spectator and not owner
		if client.IsSpectator {
			t.Error("Initial IsSpectator = true, want false")
		}
		if client.IsOwner {
			t.Error("Initial IsOwner = true, want false")
		}

		// Set as spectator
		client.IsSpectator = true
		if !client.IsSpectator {
			t.Error("IsSpectator = false after setting, want true")
		}

		// Set as owner
		client.IsOwner = true
		if !client.IsOwner {
			t.Error("IsOwner = false after setting, want true")
		}
	})
}
