package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	ToggleBacklogModeCommand struct {
		RoomID   string
		SenderID string
	}
	ToggleBacklogModeUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[ToggleBacklogModeCommand] = (*ToggleBacklogModeUseCase)(nil)

func NewToggleBacklogModeUseCase(hub domain.Hub, lockManager lock.LockManager) ToggleBacklogModeUseCase {
	return ToggleBacklogModeUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc ToggleBacklogModeUseCase) Execute(ctx context.Context, cmd ToggleBacklogModeCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.ToggleBacklogMode(ctx, cmd.SenderID); err != nil {
			return err
		}

		if err := uc.hub.SaveRoom(ctx, room); err != nil {
			return err
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return err
		}

		return nil
	})
}
