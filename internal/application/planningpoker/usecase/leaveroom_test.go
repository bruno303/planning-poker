package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
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

func TestLeaveRoomUseCase_Execute_LastLocalClient_RoomExists_DecrementsUsersAndRooms(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	roomID := "room123"
	senderID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetBus(senderID).Return(&localPresenceBus{roomID: roomID}, true)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(1)
	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls,
		expectedMetricCall{name: metric.PlanningPokerActiveUsersMetric, value: -1},
		expectedMetricCall{name: metric.PlanningPokerActiveRoomsMetric, value: -1},
	)
}

func TestLeaveRoomUseCase_Execute_NonLastLocalClient_DecrementsOnlyActiveUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	roomID := "room123"
	senderID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetBus(senderID).Return(&localPresenceBus{roomID: roomID}, true)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(2)
	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls,
		expectedMetricCall{name: metric.PlanningPokerActiveUsersMetric, value: -1},
	)
}

func TestLeaveRoomUseCase_Execute_NoLocalBus_DuplicateLeave_EmitsNoMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	roomID := "room123"
	senderID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetBus(senderID).Return(nil, false)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(0)
	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if calls := metricMeter.getCalls(); len(calls) != 0 {
		t.Fatalf("expected no metric changes for duplicate leave without local bus, got %d calls", len(calls))
	}
}

func TestLeaveRoomUseCase_Execute_WhenRoomIsMissingAfterRemove_DecrementsRoomMetricAndSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	roomID := "room123"
	senderID := "client123"

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetBus(senderID).Return(&localPresenceBus{roomID: roomID}, true)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(1)
	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls,
		expectedMetricCall{name: metric.PlanningPokerActiveUsersMetric, value: -1},
		expectedMetricCall{name: metric.PlanningPokerActiveRoomsMetric, value: -1},
	)
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

	mockHub.EXPECT().GetBus(senderID).Return(nil, false)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(0)
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
		Clients: clientcollection.New(),
	}

	expectedError := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetBus(senderID).Return(&localPresenceBus{roomID: roomID}, true)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(1)
	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
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

func TestLeaveRoomUseCase_Execute_LoadRoomErrorAfterRemove_ReturnsErrorWithoutDecrementingRooms(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	roomID := "room123"
	senderID := "client123"
	expectedError := errors.New("load failed")

	mockLockManager.EXPECT().
		ExecuteWithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) error) error {
			return fn(ctx)
		})

	mockHub.EXPECT().GetBus(senderID).Return(&localPresenceBus{roomID: roomID}, true)
	mockHub.EXPECT().GetClientsOfRoom(roomID).Return(2)
	mockHub.EXPECT().RemoveClient(ctx, senderID, roomID).Return(nil)
	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, expectedError)

	uc := NewLeaveRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := LeaveRoomCommand{
		RoomID:   roomID,
		SenderID: senderID,
	}

	err := uc.Execute(ctx, cmd)

	if !errors.Is(err, expectedError) {
		t.Fatalf("expected error %v, got %v", expectedError, err)
	}

	calls := metricMeter.getCalls()
	if countMetricCallsWithValue(calls, metric.PlanningPokerActiveUsersMetric, -1) != 1 {
		t.Fatalf("expected one active user decrement, got %d", countMetricCallsWithValue(calls, metric.PlanningPokerActiveUsersMetric, -1))
	}
	if countMetricCallsWithValue(calls, metric.PlanningPokerActiveRoomsMetric, -1) != 0 {
		t.Fatalf("expected no active room decrements, got %d", countMetricCallsWithValue(calls, metric.PlanningPokerActiveRoomsMetric, -1))
	}
}

func TestLeaveRoomUseCase_PersistenceFailureAccountsForRemovedPresenceOnce(t *testing.T) {
	for _, removed := range []bool{false, true} {
		t.Run(map[bool]string{false: "bus retained", true: "bus removed"}[removed], func(t *testing.T) {
			ctrl := gomock.NewController(t)
			hub := domain.NewMockHub(ctrl)
			manager := lock.NewMockLockManager(ctrl)
			pokerMetric, recorder := newTestPlanningPokerMetric(ctrl)
			failure := errors.New("persistence failed")
			ctx := context.Background()
			manager.EXPECT().ExecuteWithLock(gomock.Any(), "room", gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ string, fn func(context.Context) error) error { return fn(ctx) }).AnyTimes()
			gomock.InOrder(
				hub.EXPECT().GetBus("client").Return(&localPresenceBus{roomID: "room"}, true),
				hub.EXPECT().GetClientsOfRoom("room").Return(1),
				hub.EXPECT().RemoveClient(ctx, "client", "room").Return(failure),
				hub.EXPECT().GetBus("client").Return(&localPresenceBus{roomID: "room"}, !removed),
			)
			uc := NewLeaveRoomUseCase(hub, manager, pokerMetric)
			if err := uc.Execute(ctx, LeaveRoomCommand{RoomID: "room", SenderID: "client"}); !errors.Is(err, failure) {
				t.Fatalf("got %v, want persistence error", err)
			}
			if !removed {
				if len(recorder.getCalls()) != 0 {
					t.Fatal("retained presence must not decrement metrics")
				}
				return
			}
			hub.EXPECT().GetBus("client").Return(nil, false)
			hub.EXPECT().GetClientsOfRoom("room").Return(0)
			hub.EXPECT().RemoveClient(ctx, "client", "room").Return(nil)
			hub.EXPECT().LoadRoom(ctx, "room").Return(nil, domain.ErrRoomNotFound)
			if err := uc.Execute(ctx, LeaveRoomCommand{RoomID: "room", SenderID: "client"}); err != nil {
				t.Fatal(err)
			}
			assertMetricCallSequence(t, recorder.getCalls(),
				expectedMetricCall{name: metric.PlanningPokerActiveUsersMetric, value: -1},
				expectedMetricCall{name: metric.PlanningPokerActiveRoomsMetric, value: -1},
			)
		})
	}
}
