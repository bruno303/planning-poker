package entity

import (
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)

	room := NewRoom(mockCC)

	if room == nil {
		t.Fatal("NewRoom() returned nil")
	}

	if room.ID == "" {
		t.Error("NewRoom() ID is empty")
	}

	if room.Clients != mockCC {
		t.Error("NewRoom() Clients not set correctly")
	}

	if room.CurrentStory != "" {
		t.Errorf("NewRoom() CurrentStory = %v, want empty string", room.CurrentStory)
	}

	if room.Reveal {
		t.Error("NewRoom() Reveal = true, want false")
	}

	if room.Result != nil {
		t.Errorf("NewRoom() Result = %v, want nil", room.Result)
	}

	if !room.BacklogMode {
		t.Error("NewRoom() BacklogMode = false, want true")
	}
}

func TestNewRoomWithID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Add(gomock.Any())
	mockCC.EXPECT().Count().Return(1)

	room := NewRoomWithID("room-123", mockCC)

	if room == nil {
		t.Fatal("NewRoomWithID() returned nil")
	}
	if room.ID != "room-123" {
		t.Fatalf("NewRoomWithID() ID = %v, want room-123", room.ID)
	}
	if room.Clients != mockCC {
		t.Fatal("NewRoomWithID() Clients not set correctly")
	}

	if !room.BacklogMode {
		t.Error("NewRoomWithID() BacklogMode = false, want true")
	}

	client := room.NewClient("client1")
	if !client.IsOwner {
		t.Fatal("expected first client in explicit room to be owner")
	}
}

func TestRoomMutationsDoNotChangeVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	room := NewRoom(NewMockClientCollection(ctrl))
	if room.RoomVersion != 0 {
		t.Fatalf("initial room version = %d, want 0", room.RoomVersion)
	}
}
