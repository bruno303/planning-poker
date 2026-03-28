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
	"time"

	"go.uber.org/mock/gomock"
)

func assertViableRollbackCleanupContext(t *testing.T, ctx context.Context) {
	t.Helper()

	if err := ctx.Err(); err != nil {
		t.Fatalf("expected cleanup context to still be active, got %v", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected cleanup context to have a deadline")
	}

	if time.Until(deadline) <= 0 {
		t.Fatal("expected cleanup context deadline to be in the future")
	}
}

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
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
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

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())

	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
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

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls,
		expectedMetricCall{name: metric.PlanningPokerUsersTotalMetric, value: 1},
		expectedMetricCall{name: metric.PlanningPokerActiveUsersMetric, value: 1},
	)
}

func TestJoinRoomUseCase_Execute_AutoCreatesMissingRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockBus := domain.NewMockBus(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	roomID := "nonexistent"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)
	mockHub.EXPECT().NewRoomWithID(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output == nil {
		t.Fatal("expected output to be non-nil")
	}
	if output.Room != room {
		t.Fatalf("expected room %v, got %v", room, output.Room)
	}
	if output.Client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if !output.Client.IsOwner {
		t.Fatal("expected first auto-created room client to be owner")
	}

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls,
		expectedMetricCall{name: metric.PlanningPokerActiveRoomsMetric, value: 1},
		expectedMetricCall{name: metric.PlanningPokerUsersTotalMetric, value: 1},
		expectedMetricCall{name: metric.PlanningPokerActiveUsersMetric, value: 1},
	)
}

func TestJoinRoomUseCase_Execute_LoadRoomError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "room123"
	expectedError := errors.New("redis unavailable")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, expectedError)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output on load error")
	}
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected error to wrap %v, got %v", expectedError, err)
	}
	if err.Error() != "failed to load room room123: redis unavailable" {
		t.Fatalf("unexpected error message: %v", err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes on load error, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_AutoCreateRoomError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "room123"
	expectedError := errors.New("create room failed")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)
	mockHub.EXPECT().NewRoomWithID(ctx, roomID).Return(nil, expectedError)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output on auto-create error")
	}
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected error to wrap %v, got %v", expectedError, err)
	}
	if err.Error() != "failed to auto-create room room123: create room failed" {
		t.Fatalf("unexpected error message: %v", err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes on auto-create error, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_SendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
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

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(expectedError)
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Error("expected output to be nil when send fails")
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when join fails before completion, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_SendErrorOnAutoCreatedRoom_RollsBackJoinInitialization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "auto-created-room"
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

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)
	mockHub.EXPECT().NewRoomWithID(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(expectedError)
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when rollback occurs")
	}
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected error to wrap %v, got %v", expectedError, err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when auto-created room join fails before completion, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_SendErrorOnAutoCreatedRoom_DoesNotReloadRoomOrEmitMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "auto-created-room"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}
	expectedError := errors.New("send failed")
	loadError := errors.New("load room failed")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)
	mockHub.EXPECT().NewRoomWithID(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(expectedError)
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when rollback occurs")
	}
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected error to wrap %v, got %v", expectedError, err)
	}
	if errors.Is(err, loadError) {
		t.Fatalf("expected rollback cleanup to ignore room reload error, got %v", err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when auto-created room join fails before completion, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_SendErrorWhenRollbackCleanupFails_DoesNotDecrementMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "auto-created-room"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}
	sendErr := errors.New("send failed")
	removeErr := errors.New("remove client failed")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)
	mockHub.EXPECT().NewRoomWithID(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(sendErr)
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).Return(removeErr)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when rollback cleanup fails")
	}
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected error to wrap %v, got %v", sendErr, err)
	}
	if !errors.Is(err, removeErr) {
		t.Fatalf("expected error to wrap %v, got %v", removeErr, err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when join fails before completion, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_BroadcastErrorAfterSend_RollsBackWithoutEmittingMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "room123"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}
	broadcastErr := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(broadcastErr)
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when broadcast fails")
	}
	if !errors.Is(err, broadcastErr) {
		t.Fatalf("expected error to wrap %v, got %v", broadcastErr, err)
	}

	calls := metricMeter.getCalls()
	if got := countMetricCalls(calls, metric.PlanningPokerActiveRoomsMetric); got != 0 {
		t.Fatalf("expected no %q metric emissions, got %d", metric.PlanningPokerActiveRoomsMetric, got)
	}
	if got := countMetricCalls(calls, metric.PlanningPokerUsersTotalMetric); got != 0 {
		t.Fatalf("expected no %q metric emissions, got %d", metric.PlanningPokerUsersTotalMetric, got)
	}
	if got := countMetricCalls(calls, metric.PlanningPokerActiveUsersMetric); got != 0 {
		t.Fatalf("expected no %q metric emissions, got %d", metric.PlanningPokerActiveUsersMetric, got)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when broadcast fails after send, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_BroadcastErrorAfterSendOnAutoCreatedRoom_DoesNotEmitMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
	mockBus := domain.NewMockBus(ctrl)

	roomID := "auto-created-room"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(),
	}
	broadcastErr := errors.New("broadcast failed")

	mockLockManager.EXPECT().
		WithLock(gomock.Any(), roomID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		})

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)
	mockHub.EXPECT().NewRoomWithID(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(broadcastErr)
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).Return(nil)

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when broadcast fails")
	}
	if !errors.Is(err, broadcastErr) {
		t.Fatalf("expected error to wrap %v, got %v", broadcastErr, err)
	}

	calls := metricMeter.getCalls()
	if got := countMetricCalls(calls, metric.PlanningPokerActiveRoomsMetric); got != 0 {
		t.Fatalf("expected no %q metric emissions, got %d", metric.PlanningPokerActiveRoomsMetric, got)
	}
	if got := countMetricCalls(calls, metric.PlanningPokerUsersTotalMetric); got != 0 {
		t.Fatalf("expected no %q metric emissions, got %d", metric.PlanningPokerUsersTotalMetric, got)
	}
	if got := countMetricCalls(calls, metric.PlanningPokerActiveUsersMetric); got != 0 {
		t.Fatalf("expected no %q metric emissions, got %d", metric.PlanningPokerActiveUsersMetric, got)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when auto-created room broadcast fails after send, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_SendFailsFromCanceledContext_UsesActiveRollbackCleanupContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
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

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sendCtx context.Context, _ any) error {
		cancel()
		return sendCtx.Err()
	})
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).DoAndReturn(func(cleanupCtx context.Context, _ string, _ string) error {
		assertViableRollbackCleanupContext(t, cleanupCtx)
		return nil
	})

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when send fails")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to wrap %v, got %v", context.Canceled, err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when join fails before completion, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_BroadcastFailsFromCanceledContext_UsesActiveRollbackCleanupContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	mockHub := domain.NewMockHub(ctrl)
	mockLockManager := lock.NewMockLockManager(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
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

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().AddClient(gomock.Any())
	mockHub.EXPECT().AddBus(gomock.Any(), gomock.Any(), gomock.Any())
	mockBus.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, any) error {
		cancel()
		return nil
	})
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).DoAndReturn(func(broadcastCtx context.Context, _ string, _ any) error {
		if !errors.Is(broadcastCtx.Err(), context.Canceled) {
			t.Fatalf("expected broadcast context to be canceled, got %v", broadcastCtx.Err())
		}
		return broadcastCtx.Err()
	})
	mockHub.EXPECT().RemoveClient(gomock.Any(), gomock.Any(), roomID).DoAndReturn(func(cleanupCtx context.Context, _ string, _ string) error {
		assertViableRollbackCleanupContext(t, cleanupCtx)
		return nil
	})

	uc := NewJoinRoomUseCase(mockHub, mockLockManager, testMetric)
	cmd := JoinRoomCommand{
		RoomID:   roomID,
		SenderID: "sender123",
		Bus:      mockBus,
	}

	output, err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if output != nil {
		t.Fatal("expected nil output when broadcast fails")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to wrap %v, got %v", context.Canceled, err)
	}

	calls := metricMeter.getCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no metric changes when broadcast fails after send, got %d calls", len(calls))
	}
}

func TestJoinRoomUseCase_Execute_NilDependencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetric := metric.NewPlanningPokerMetric()

	tests := []struct {
		name string
		hub  domain.Hub
		lock lock.LockManager
	}{
		{name: "nil hub", hub: nil, lock: lock.NewMockLockManager(ctrl)},
		{name: "nil lockmanager", hub: domain.NewMockHub(ctrl), lock: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expected panic, but none occurred")
				}
			}()

			_ = NewJoinRoomUseCase(tc.hub, tc.lock, mockMetric)
		})
	}
}
