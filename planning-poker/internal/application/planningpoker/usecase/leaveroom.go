package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	LeaveRoomCommand struct {
		RoomID   string
		SenderID string
	}
	LeaveRoomUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

func NewLeaveRoomUseCase(hub domain.Hub, lockManager lock.LockManager) LeaveRoomUseCase {
	return LeaveRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc LeaveRoomUseCase) Execute(ctx context.Context, cmd LeaveRoomCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		if err := uc.hub.RemoveClient(ctx, cmd.SenderID, cmd.RoomID); err != nil {
			return err
		}

		// if room still exists, broadcast the updated state
		if room, ok := uc.hub.GetRoom(cmd.RoomID); ok {
			if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
				return err
			}
		}

		return nil
	})
}
