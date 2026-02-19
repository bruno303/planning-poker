package usecase

import (
	"context"
	"planning-poker/internal/application"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"

	"github.com/bruno303/go-toolkit/pkg/log"
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
		logger      log.Logger
	}
)

var _ application.UseCase[LeaveRoomCommand] = (*leaveRoomUseCase)(nil)

func NewLeaveRoomUseCase(hub domain.Hub, lockManager lock.LockManager, metric metric.PlanningPokerMetric) *leaveRoomUseCase {
	return &leaveRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
		metric:      metric,
		logger:      log.NewLogger("usecase.leaveroom"),
	}
}

func (uc *leaveRoomUseCase) Execute(ctx context.Context, cmd LeaveRoomCommand) error {
	uc.logger.Info(ctx, "Client %s leaving room %s", cmd.SenderID, cmd.RoomID)
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		if err := uc.hub.RemoveClient(ctx, cmd.SenderID, cmd.RoomID); err != nil {
			uc.logger.Error(ctx, "Error removing client from room", err)
			return err
		}

		uc.metric.DecrementActiveUsers(ctx)

		// if room still exists, broadcast the updated state
		// otherwise, decrement active rooms metric
		if room, ok := uc.hub.GetRoom(ctx, cmd.RoomID); ok {
			if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
				uc.logger.Error(ctx, "Error broadcasting room state", err)
				return err
			}
		} else {
			uc.metric.DecrementActiveRoomsCounter(ctx)
		}

		uc.logger.Info(ctx, "Client %s left room %s successfully", cmd.SenderID, cmd.RoomID)
		return nil
	})
}
