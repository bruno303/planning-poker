package usecase

import (
	"context"
	"fmt"
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

func NewToggleOwnerUseCase(hub domain.Hub, lockManager lock.LockManager) ToggleOwnerUseCase {
	return ToggleOwnerUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc ToggleOwnerUseCase) Execute(ctx context.Context, cmd ToggleOwnerCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, ok := uc.hub.GetRoom(ctx, cmd.RoomID)
		if !ok {
			return fmt.Errorf("room %s not found", cmd.RoomID)
		}

		if err := room.ToggleOwner(ctx, cmd.SenderID, cmd.TargetClientID); err != nil {
			return err
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return err
		}

		return nil
	})
}
