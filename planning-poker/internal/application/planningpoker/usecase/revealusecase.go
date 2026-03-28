package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	RevealCommand struct {
		RoomID   string
		SenderID string
	}
	RevealUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[RevealCommand] = (*RevealUseCase)(nil)

func NewRevealUseCase(hub domain.Hub, lockManager lock.LockManager) RevealUseCase {
	return RevealUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc RevealUseCase) Execute(ctx context.Context, cmd RevealCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.ToggleReveal(ctx, cmd.SenderID); err != nil {
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
