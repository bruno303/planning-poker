package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/clientcollection"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewVoteUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	uc := NewVoteUseCase(mockHub, mockLockManager)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
}

func TestVoteUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	senderID := "client123"
	vote := "5"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	room.NewClient(senderID)

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().SaveRoom(ctx, room).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewVoteUseCase(mockHub, mockLockManager)
	cmd := VoteCommand{
		RoomID:   roomID,
		SenderID: senderID,
		Vote:     &vote,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVoteUseCase_Execute_SaveRoomError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	senderID := "client123"
	vote := "5"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	room.NewClient(senderID)
	expectedError := errors.New("failed to save room")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().SaveRoom(ctx, room).Return(expectedError)

	uc := NewVoteUseCase(mockHub, mockLockManager)
	cmd := VoteCommand{
		RoomID:   roomID,
		SenderID: senderID,
		Vote:     &vote,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

func TestVoteUseCase_Execute_RoomNotFound(t *testing.T) {
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

	uc := NewVoteUseCase(mockHub, mockLockManager)
	cmd := VoteCommand{
		RoomID:   roomID,
		SenderID: "client123",
		Vote:     nil,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "room nonexistent not found" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}

func TestVoteUseCase_Execute_BroadcastError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)

	roomID := "room123"
	senderID := "client123"
	vote := "5"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	room.NewClient(senderID)
	expectedError := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().SaveRoom(ctx, room).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(expectedError)

	uc := NewVoteUseCase(mockHub, mockLockManager)
	cmd := VoteCommand{
		RoomID:   roomID,
		SenderID: senderID,
		Vote:     &vote,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}
