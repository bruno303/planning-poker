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
	unlockPresence, err := lockLocalPresence(ctx)
	if err != nil {
		return err
	}
	defer unlockPresence()

	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		// A late leave from an old room must not count the replacement bus.
		hadBusInRoom := domain.HasBusInRoom(uc.hub, cmd.SenderID, cmd.RoomID)
		localClientsBefore := uc.hub.GetClientsOfRoom(cmd.RoomID)

		removeErr := uc.hub.RemoveClient(ctx, cmd.SenderID, cmd.RoomID)
		// Redis can remove local presence before a persistence failure. Account
		// for that removal now so a retry cannot lose or duplicate the delta.
		removedLocalBus := hadBusInRoom
		if removeErr != nil && hadBusInRoom {
			removedLocalBus = !domain.HasBusInRoom(uc.hub, cmd.SenderID, cmd.RoomID)
		}

		if removedLocalBus {
			uc.metric.DecrementActiveUsers(ctx)
		}
		if removedLocalBus && localClientsBefore == 1 {
			uc.metric.DecrementActiveRoomsCounter(ctx)
		}
		if removeErr != nil {
			uc.logger.Error(ctx, "Error removing client from room", removeErr)
			return removeErr
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
