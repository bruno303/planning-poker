package usecase

import (
	"context"
	"testing"

	"planning-poker/internal/application/lock"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"go.uber.org/mock/gomock"
)

func TestSelectStoryUseCaseExecute(t *testing.T) {
	ctrl := gomock.NewController(t)
	room := selectStoryTestRoom()
	hub := domain.NewMockHub(ctrl)
	lockManager := lock.NewMockLockManager(ctrl)
	ctx := context.Background()

	lockManager.EXPECT().ExecuteWithLock(gomock.Any(), room.ID, gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ string, fn func(context.Context) error) error { return fn(ctx) },
	)
	hub.EXPECT().LoadRoom(ctx, room.ID).Return(room, nil)
	hub.EXPECT().SaveRoomIfVersion(ctx, room, gomock.Any()).Return(nil)
	hub.EXPECT().BroadcastToRoom(ctx, room.ID, gomock.Any()).Return(nil)

	err := NewSelectStoryUseCase(hub, lockManager).Execute(ctx, SelectStoryCommand{
		RoomID: room.ID, SenderID: "owner", StoryID: "pending",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if room.CurrentStoryIndex != 1 {
		t.Fatalf("current story index = %d, want 1", room.CurrentStoryIndex)
	}
}

func TestSelectStoryUseCaseExecuteValidationFailureDoesNotPersist(t *testing.T) {
	ctrl := gomock.NewController(t)
	room := selectStoryTestRoom()
	hub := domain.NewMockHub(ctrl)
	lockManager := lock.NewMockLockManager(ctrl)
	ctx := context.Background()

	lockManager.EXPECT().ExecuteWithLock(gomock.Any(), room.ID, gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ string, fn func(context.Context) error) error { return fn(ctx) },
	)
	hub.EXPECT().LoadRoom(ctx, room.ID).Return(room, nil)
	hub.EXPECT().SaveRoomIfVersion(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	hub.EXPECT().BroadcastToRoom(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	err := NewSelectStoryUseCase(hub, lockManager).Execute(ctx, SelectStoryCommand{
		RoomID: room.ID, SenderID: "owner", StoryID: "missing",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func selectStoryTestRoom() *entity.Room {
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
