package usecase

import (
	"context"
	"errors"
	"testing"

	"planning-poker/internal/application/lock"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"go.uber.org/mock/gomock"
)

func TestReorderStoryUseCaseExecute(t *testing.T) {
	ctrl := gomock.NewController(t)
	room := reorderStoryTestRoom()
	hub := domain.NewMockHub(ctrl)
	lockManager := lock.NewMockLockManager(ctrl)
	ctx := context.Background()

	lockManager.EXPECT().ExecuteWithLock(gomock.Any(), room.ID, gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ string, fn func(context.Context) error) error { return fn(ctx) },
	)
	hub.EXPECT().LoadRoom(ctx, room.ID).Return(room, nil)
	hub.EXPECT().SaveRoom(ctx, room).Return(nil)
	hub.EXPECT().BroadcastToRoom(ctx, room.ID, gomock.Any()).Return(nil)

	err := NewReorderStoryUseCase(hub, lockManager).Execute(ctx, ReorderStoryCommand{
		RoomID: room.ID, SenderID: "owner", StoryID: "pending", TargetIndex: 0,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if room.Stories[0].ID != "pending" {
		t.Fatalf("unexpected reordered room: %+v", room)
	}
}

func TestReorderStoryUseCaseExecuteLoadFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	hub := domain.NewMockHub(ctrl)
	lockManager := lock.NewMockLockManager(ctrl)
	ctx := context.Background()
	roomID := "missing-room"
	expectedErr := errors.New("room unavailable")

	lockManager.EXPECT().ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ string, fn func(context.Context) error) error { return fn(ctx) },
	)
	hub.EXPECT().LoadRoom(ctx, roomID).Return(nil, expectedErr)

	err := NewReorderStoryUseCase(hub, lockManager).Execute(ctx, ReorderStoryCommand{RoomID: roomID})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func reorderStoryTestRoom() *entity.Room {
	room := &entity.Room{
		ID:                "room-1",
		Clients:           clientcollection.New(),
		BacklogMode:       true,
		Stories:           []entity.Story{{ID: "current", Name: "Current"}, {ID: "pending", Name: "Pending"}},
		CurrentStoryIndex: 0,
	}
	owner := room.NewClient("owner")
	owner.IsOwner = true
	return room
}
