package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewToggleOwnerUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	uc := NewToggleOwnerUseCase(mockHub, mockLockManager)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
}

func TestToggleOwnerUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	senderID := "sender123"
	targetID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	sender := room.NewClient(senderID)
	sender.IsOwner = true
	target := room.NewClient(targetID)
	target.IsOwner = false

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := ToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: targetID,
		SenderID:       senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestToggleOwnerUseCase_Execute_RoomNotFound(t *testing.T) {
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

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(nil, false)

	uc := NewToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := ToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: "client123",
		SenderID:       "sender123",
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "room nonexistent not found" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}

func TestToggleOwnerUseCase_Execute_BroadcastError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	senderID := "sender123"
	targetID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	sender := room.NewClient(senderID)
	sender.IsOwner = true
	target := room.NewClient(targetID)
	target.IsOwner = false
	expectedError := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(expectedError)

	uc := NewToggleOwnerUseCase(mockHub, mockLockManager)
	cmd := ToggleOwnerCommand{
		RoomID:         roomID,
		TargetClientID: targetID,
		SenderID:       senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}
