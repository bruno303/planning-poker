package entity_test

import (
	"context"
	"errors"
	"testing"

	"planning-poker/internal/domain/domainerror"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
)

func assertAdminToggleOwnerResult(t *testing.T, room *entity.Room, targetID string, wantIsOwner, wantErrLastOwner bool, wantErrSentinel, wantErr error) {
	t.Helper()

	err := room.AdminToggleOwner(context.Background(), targetID)
	if wantErrLastOwner {
		if err == nil || !errors.Is(err, domainerror.ErrLastOwner) {
			t.Errorf("expected ErrLastOwner, got %v", err)
		}
		return
	}
	if wantErrSentinel != nil {
		if err == nil || !errors.Is(err, wantErrSentinel) {
			t.Errorf("expected sentinel %v, got %v", wantErrSentinel, err)
		}
		return
	}
	if wantErr != nil {
		if err == nil || err.Error() != wantErr.Error() {
			t.Errorf("error = %v, want %v", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	client, ok := room.FindClient(targetID)
	if !ok {
		t.Fatal("target client not found after toggle")
	}
	if client.IsOwner != wantIsOwner {
		t.Errorf("IsOwner = %v, want %v", client.IsOwner, wantIsOwner)
	}
}

func TestRoom_AdminToggleOwner(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(room *entity.Room) string // returns targetClientID
		wantIsOwner      bool
		wantErr          error
		wantErrLastOwner bool
		wantErrSentinel  error
	}{
		{
			name: "grant owner to non-owner",
			setup: func(room *entity.Room) string {
				room.NewClient("owner1").IsOwner = true
				target := room.NewClient("target1")
				return target.ID
			},
			wantIsOwner: true,
		},
		{
			name: "revoke owner when multiple owners exist",
			setup: func(room *entity.Room) string {
				room.NewClient("owner1").IsOwner = true
				target := room.NewClient("owner2")
				target.IsOwner = true
				return target.ID
			},
			wantIsOwner: false,
		},
		{
			name: "refuse revoke from last owner",
			setup: func(room *entity.Room) string {
				target := room.NewClient("owner1")
				target.IsOwner = true
				room.NewClient("client1")
				return target.ID
			},
			wantIsOwner:      true,
			wantErrLastOwner: true,
		},
		{
			name: "target not found",
			setup: func(room *entity.Room) string {
				return "nonexistent"
			},
			wantErrSentinel: domainerror.ErrClientNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := &entity.Room{ID: "room1", Clients: clientcollection.New()}
			targetID := tt.setup(room)

			assertAdminToggleOwnerResult(t, room, targetID, tt.wantIsOwner, tt.wantErrLastOwner, tt.wantErrSentinel, tt.wantErr)
		})
	}
}
