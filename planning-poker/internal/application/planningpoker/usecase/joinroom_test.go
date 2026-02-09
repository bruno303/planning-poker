package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/clientcollection"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewJoinRoomUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, mockMetric)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
	if uc.lockManager != mockLockManager {
		t.Error("lockManager not set correctly")
	}
}

func TestJoinRoomUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()
	mockBus := domain.NewMockBus(ctrl)

	roomID := "room123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())

	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	busFactory := func(clientID string) domain.Bus {
		return mockBus
	}

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := JoinRoomCommand{
		RoomID:     roomID,
		SenderID:   "sender123",
		BusFactory: busFactory,
	}

	output, err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output == nil {
		t.Fatal("expected output to be non-nil")
	}
	if output.Client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if output.Room != room {
		t.Errorf("expected room %v, got %v", room, output.Room)
	}
	if output.Bus != mockBus {
		t.Error("expected bus to match mockBus")
	}
}

func TestJoinRoomUseCase_Execute_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	roomID := "nonexistent"

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(nil, false)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		BusFactory: func(clientID string) domain.Bus {
			return nil
		},
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "room nonexistent not found" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
	if output != nil {
		t.Error("expected output to be nil on error")
	}
}

func TestJoinRoomUseCase_Execute_SendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()
	mockBus := domain.NewMockBus(ctrl)

	roomID := "room123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	expectedError := errors.New("send failed")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(expectedError)

	busFactory := func(clientID string) domain.Bus {
		return mockBus
	}

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, mockMetric)
	cmd := JoinRoomCommand{
		RoomID:     roomID,
		SenderID:   "sender123",
		BusFactory: busFactory,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Error("expected output to be nil when send fails")
	}
}
