package usecase

import (
	"context"
	"fmt"
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

func NewToggleSpectatorUseCase(hub domain.Hub, lockManager lock.LockManager) ToggleSpectatorUseCase {
	return ToggleSpectatorUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc ToggleSpectatorUseCase) Execute(ctx context.Context, cmd ToggleSpectatorCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, ok := uc.hub.GetRoom(ctx, cmd.RoomID)
		if !ok {
			return fmt.Errorf("room %s not found", cmd.RoomID)
		}

		if err := room.ToggleSpectator(ctx, cmd.SenderID, cmd.TargetClientID); err != nil {
			return err
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return err
		}

		return nil
	})
}
