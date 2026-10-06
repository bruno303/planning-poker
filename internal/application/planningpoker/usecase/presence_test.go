package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"

	"go.uber.org/mock/gomock"
)

func TestQueuedPresenceTransitionDoesNotAcquireRoomLock(t *testing.T) {
	for _, operation := range []string{"join", "leave"} {
		t.Run(operation, func(t *testing.T) {
			unlock, err := lockLocalPresence(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer unlock()
			ctrl := gomock.NewController(t)
			// No hub or room-lock calls are allowed while the presence guard is held.
			hub := domain.NewMockHub(ctrl)
			manager := lock.NewMockLockManager(ctrl)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			if operation == "join" {
				join := NewJoinRoomUseCase(hub, manager, metric.NewPlanningPokerMetric())
				_, err = join.Execute(ctx, JoinRoomCommand{RoomID: "queued-room"})
			} else {
				leave := NewLeaveRoomUseCase(hub, manager, metric.NewPlanningPokerMetric())
				err = leave.Execute(ctx, LeaveRoomCommand{RoomID: "queued-room"})
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("got %v, want deadline exceeded", err)
			}
		})
	}
}
