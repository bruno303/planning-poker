package usecase

import (
	"context"
	"errors"
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

var _ UseCase[LeaveRoomCommand] = (*leaveRoomUseCase)(nil)

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
		// The client stops being an active local user only when the bus being
		// removed is the one registered for this room. When the same client has
		// already moved to another room on this instance, the join side kept the
		// active-user counter, and the late leave of the old socket must not
		// decrement it again. localClientsBefore decides whether this instance must
		// also drop the room.
		hadBusInRoom := uc.hub.HasBusInRoom(cmd.SenderID, cmd.RoomID)
		localClientsBefore := uc.hub.GetClientsOfRoom(cmd.RoomID)

		if err := uc.hub.RemoveClient(ctx, cmd.SenderID, cmd.RoomID); err != nil {
			uc.logger.Error(ctx, "Error removing client from room", err)
			return err
		}

		if hadBusInRoom {
			uc.metric.DecrementActiveUsers(ctx)
		}
		if localClientsBefore > 0 && localClientsBefore <= 1 {
			uc.metric.DecrementActiveRoomsCounter(ctx)
		}

		// if room still exists, broadcast the updated state
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err == nil {
			if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
				uc.logger.Error(ctx, "Error broadcasting room state", err)
				return err
			}
		} else if !errors.Is(err, domain.ErrRoomNotFound) {
			uc.logger.Error(ctx, "Error loading room after client removal", err)
			return err
		}

		uc.logger.Info(ctx, "Client %s left room %s successfully", cmd.SenderID, cmd.RoomID)
		return nil
	})
}
