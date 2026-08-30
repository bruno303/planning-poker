package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestAdminToggleOwnerUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	targetID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	owner := room.NewClient("owner123")
	owner.IsOwner = true
	target := room.NewClient(targetID)
	target.IsOwner = false

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().SaveRoom(ctx, room).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewAdminToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := AdminToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: targetID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAdminToggleOwnerUseCase_Execute_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "nonexistent"

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)

	uc := NewAdminToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := AdminToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: "client123",
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrRoomNotFound) {
		t.Errorf("expected ErrRoomNotFound, got %v", err)
	}
}

func TestAdminToggleOwnerUseCase_Execute_SaveRoomError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	targetID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	owner := room.NewClient("owner123")
	owner.IsOwner = true
	room.NewClient(targetID)
	expectedError := errors.New("failed to save room")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().SaveRoom(ctx, room).Return(expectedError)

	uc := NewAdminToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := AdminToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: targetID,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedError) {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

func TestAdminToggleOwnerUseCase_Execute_BroadcastError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	targetID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	owner := room.NewClient("owner123")
	owner.IsOwner = true
	target := room.NewClient(targetID)
	target.IsOwner = false
	expectedError := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().SaveRoom(ctx, room).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(expectedError)

	uc := NewAdminToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := AdminToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: targetID,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

func TestAdminToggleOwnerUseCase_Execute_LastOwnerRefused(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	targetID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	// Only one client in the room, and they are the owner — so AdminToggleOwner returns ErrLastOwner
	client := room.NewClient(targetID)
	client.IsOwner = true

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	// SaveRoom and BroadcastToRoom must NOT be called (lock function returns early)

	uc := NewAdminToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := AdminToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: targetID,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrLastOwner) {
		t.Errorf("expected ErrLastOwner, got %v", err)
	}
}
