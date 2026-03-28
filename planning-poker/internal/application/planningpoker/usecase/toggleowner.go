package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	ToggleOwnerCommand struct {
		RoomID         string
		SenderID       string
		TargetClientID string
	}
	ToggleOwnerUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[ToggleOwnerCommand] = (*ToggleOwnerUseCase)(nil)

func NewToggleOwnerUseCase(hub domain.Hub, lockManager lock.LockManager) ToggleOwnerUseCase {
	return ToggleOwnerUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc ToggleOwnerUseCase) Execute(ctx context.Context, cmd ToggleOwnerCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.ToggleOwner(ctx, cmd.SenderID, cmd.TargetClientID); err != nil {
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
