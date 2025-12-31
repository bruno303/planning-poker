package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewLeaveRoomUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, mockMetric)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
	if uc.lockManager != mockLockManager {
		t.Error("lockManager not set correctly")
	}
}

func TestLeaveRoomUseCase_Execute_Success_RoomExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	roomID := "room123"
	senderID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}


	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLeaveRoomUseCase_Execute_Success_RoomRemoved(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	roomID := "room123"
	senderID := "client123"

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().GetRoom(ctx, roomID).Return(nil, false)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLeaveRoomUseCase_Execute_RemoveClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	roomID := "room123"
	senderID := "client123"
	expectedError := errors.New("remove client failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(expectedError)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

func TestLeaveRoomUseCase_Execute_BroadcastError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	roomID := "room123"
	senderID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	expectedError := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(expectedError)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}
