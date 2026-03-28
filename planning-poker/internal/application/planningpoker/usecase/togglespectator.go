package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	ToggleSpectatorCommand struct {
		RoomID         string
		SenderID       string
		TargetClientID string
	}
	ToggleSpectatorUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[ToggleSpectatorCommand] = (*ToggleSpectatorUseCase)(nil)

func NewToggleSpectatorUseCase(hub domain.Hub, lockManager lock.LockManager) ToggleSpectatorUseCase {
	return ToggleSpectatorUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc ToggleSpectatorUseCase) Execute(ctx context.Context, cmd ToggleSpectatorCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.ToggleSpectator(ctx, cmd.SenderID, cmd.TargetClientID); err != nil {
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
