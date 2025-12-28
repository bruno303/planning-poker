package usecase

import (
	"context"
	"planning-poker/internal/application"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	LeaveRoomCommand struct {
		RoomID   string
		SenderID string
	}
	leaveRoomUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
		metric      metric.PlanningPokerMetric
	}
)

var _ application.UseCase[LeaveRoomCommand] = (*leaveRoomUseCase)(nil)

func NewLeaveRoomUseCase(hub domain.Hub, lockManager lock.LockManager, metric metric.PlanningPokerMetric) *leaveRoomUseCase {
	return &leaveRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
		metric:      metric,
	}
}

func (uc *leaveRoomUseCase) Execute(ctx context.Context, cmd LeaveRoomCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		if err := uc.hub.RemoveClient(ctx, cmd.SenderID, cmd.RoomID); err != nil {
			return err
		}

		uc.metric.DecrementActiveUsers(ctx)

		// if room still exists, broadcast the updated state
		// otherwise, decrement active rooms metric
		if room, ok := uc.hub.GetRoom(ctx, cmd.RoomID); ok {
			if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
				return err
			}
		} else {
			uc.metric.DecrementActiveRoomsCounter(ctx)
		}

		return nil
	})
}
