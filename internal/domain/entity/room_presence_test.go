package entity

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestRoom_NewClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Add(gomock.Any()).Times(3)
	mockCC.EXPECT().Count().Return(1).Times(1) // First client
	mockCC.EXPECT().Count().Return(2).Times(1) // Second client
	mockCC.EXPECT().Count().Return(3).Times(1) // Third client

	room := NewRoom(mockCC)

	// First client should be owner
	client1 := room.NewClient("client1")
	if client1 == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client1.ID != "client1" {
		t.Errorf("NewClient() ID = %v, want client1", client1.ID)
	}
	if !client1.IsOwner {
		t.Error("First client should be owner")
	}

	// Second client should not be owner
	client2 := room.NewClient("client2")
	if client2.IsOwner {
		t.Error("Second client should not be owner")
	}

	// Third client should not be owner
	client3 := room.NewClient("client3")
	if client3.IsOwner {
		t.Error("Third client should not be owner")
	}
}

func TestRoom_RemoveClient(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove client and reassign owner if needed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client2 := &Client{ID: cid("client2"), IsOwner: false, IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Remove("client1")
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).AnyTimes()
		mockCC.EXPECT().Count().Return(0).Times(1) // After removal, 0 owners
		mockCC.EXPECT().Count().Return(1).Times(1) // But 1 client remains
		mockCC.EXPECT().First().Return(client2, true)
		mockCC.EXPECT().Values().Return([]*Client{client2}).AnyTimes()

		room := NewRoom(mockCC)

		err := room.RemoveClient(ctx, cid("client1"))
		if err != nil {
			t.Errorf("RemoveClient() error = %v", err)
		}

		// client2 should now be owner
		if !client2.IsOwner {
			t.Error("Expected client2 to become owner after removing the only owner")
		}
	})

	t.Run("should remove client and not reassign if other owners exist", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client2 := &Client{ID: cid("client2"), IsOwner: true, IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Remove("client1")
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).AnyTimes()
		mockCC.EXPECT().Count().Return(1).Times(1) // 1 owner remains
		mockCC.EXPECT().Values().Return([]*Client{client2}).AnyTimes()

		room := NewRoom(mockCC)

		err := room.RemoveClient(ctx, cid("client1"))
		if err != nil {
			t.Errorf("RemoveClient() error = %v", err)
		}
	})
}

func TestRoom_CountOwners(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)
	mockFilteredCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockFilteredCC)
	mockFilteredCC.EXPECT().Count().Return(2)

	room := NewRoom(mockCC)

	count := room.CountOwners()
	if count != 2 {
		t.Errorf("CountOwners() = %v, want 2", count)
	}
}

func TestRoom_ToggleSpectator(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle spectator status when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: cid("owner"), IsOwner: true, IsSpectator: false}
		target := &Client{ID: cid("target"), IsOwner: false, IsSpectator: false, HasVoted: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(3) // getClient twice + checkReveal
		mockCC.EXPECT().First().Return(owner, true).Times(1)
		mockCC.EXPECT().First().Return(target, true).Times(1)
		mockCC.EXPECT().Values().Return([]*Client{owner, target}).AnyTimes()

		room := NewRoom(mockCC)

		err := room.ToggleSpectator(ctx, cid("owner"), cid("target"))
		if err != nil {
			t.Errorf("ToggleSpectator() error = %v", err)
		}

		if !target.IsSpectator {
			t.Error("ToggleSpectator() should toggle spectator status to true")
		}

		// Toggle back
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(3)
		mockCC.EXPECT().First().Return(owner, true).Times(1)
		mockCC.EXPECT().First().Return(target, true).Times(1)

		err = room.ToggleSpectator(ctx, cid("owner"), cid("target"))
		if err != nil {
			t.Errorf("ToggleSpectator() error = %v", err)
		}

		if target.IsSpectator {
			t.Error("ToggleSpectator() should toggle spectator status back to false")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: cid("client1"), IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.ToggleSpectator(ctx, cid("client1"), cid("target"))
		if err == nil {
			t.Error("ToggleSpectator() expected error for non-owner")
		}
	})

	t.Run("should fail when target client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: cid("owner"), IsOwner: true}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(2)
		mockCC.EXPECT().First().Return(owner, true).Times(1)
		mockCC.EXPECT().First().Return(nil, false).Times(1)

		room := NewRoom(mockCC)

		err := room.ToggleSpectator(ctx, cid("owner"), cid("nonexistent"))
		if err == nil {
			t.Error("ToggleSpectator() expected error for nonexistent target")
		}
	})
}

func TestRoom_ToggleOwner(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle owner status when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: cid("owner"), IsOwner: true}
		target := &Client{ID: cid("target"), IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockFilteredCC := NewMockClientCollection(ctrl)

		// First getClient for owner
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(owner, true).Times(1)

		// Filter for owners
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockFilteredCC).Times(1)
		mockFilteredCC.EXPECT().Count().Return(2) // 2 owners exist

		// Second getClient for target
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(target, true).Times(1)

		room := NewRoom(mockCC)

		err := room.ToggleOwner(ctx, cid("owner"), cid("target"))
		if err != nil {
			t.Errorf("ToggleOwner() error = %v", err)
		}

		if !target.IsOwner {
			t.Error("ToggleOwner() should toggle owner status to true")
		}
	})

	t.Run("should not remove last owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: cid("owner"), IsOwner: true}

		mockCC := NewMockClientCollection(ctrl)
		mockFilteredCC := NewMockClientCollection(ctrl)

		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(owner, true).Times(1)

		mockCC.EXPECT().Filter(gomock.Any()).Return(mockFilteredCC).Times(1)
		mockFilteredCC.EXPECT().Count().Return(1) // Only 1 owner
		mockFilteredCC.EXPECT().First().Return(owner, true)

		room := NewRoom(mockCC)

		err := room.ToggleOwner(ctx, cid("owner"), cid("owner"))
		if err != nil {
			t.Errorf("ToggleOwner() error = %v", err)
		}

		if !owner.IsOwner {
			t.Error("ToggleOwner() should not remove last owner")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: cid("client1"), IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.ToggleOwner(ctx, cid("client1"), cid("target"))
		if err == nil {
			t.Error("ToggleOwner() expected error for non-owner")
		}
	})
}

func TestRoom_IsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)

	t.Run("should return true when room is empty", func(t *testing.T) {
		mockCC.EXPECT().Count().Return(0)
		room := NewRoom(mockCC)

		if !room.IsEmpty() {
			t.Error("IsEmpty() should return true for empty room")
		}
	})

	t.Run("should return false when room has clients", func(t *testing.T) {
		mockCC.EXPECT().Count().Return(3)
		room := NewRoom(mockCC)

		if room.IsEmpty() {
			t.Error("IsEmpty() should return false for non-empty room")
		}
	})
}

func TestRoom_UpdateClientName(t *testing.T) {
	ctx := context.Background()

	t.Run("should update client name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: cid("client1"), Name: "Old Name"}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.UpdateClientName(ctx, cid("client1"), "New Name")
		if err != nil {
			t.Errorf("UpdateClientName() error = %v", err)
		}

		if client.Name != "New Name" {
			t.Errorf("UpdateClientName() Name = %v, want 'New Name'", client.Name)
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nil, false)

		room := NewRoom(mockCC)

		err := room.UpdateClientName(ctx, cid("nonexistent"), "New Name")
		if err == nil {
			t.Error("UpdateClientName() expected error for nonexistent client")
		}
	})
}
