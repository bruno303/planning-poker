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

func TestNewVoteAgainUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	uc := NewVoteAgainUseCase(mockHub, mockLockManager)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
}

func TestVoteAgainUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	clientID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	client := room.NewClient(clientID)
	client.IsOwner = true

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewVoteAgainUseCase(mockHub, mockLockManager)
	cmd := VoteAgainCommand{
		RoomID:   roomID,
		SenderID: "client123",
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVoteAgainUseCase_Execute_RoomNotFound(t *testing.T) {
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

	uc := NewVoteAgainUseCase(mockHub, mockLockManager)
	cmd := VoteAgainCommand{
		RoomID:   roomID,
		SenderID: "client123",
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "room nonexistent not found" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}

func TestVoteAgainUseCase_Execute_BroadcastError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	clientID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	client := room.NewClient(clientID)
	client.IsOwner = true
	expectedError := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(expectedError)

	uc := NewVoteAgainUseCase(mockHub, mockLockManager)
	cmd := VoteAgainCommand{
		RoomID:   roomID,
		SenderID: "client123",
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}
