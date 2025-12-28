package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	ResetCommand struct {
		RoomID   string
		SenderID string
	}
	ResetUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ application.UseCase[ResetCommand] = (*ResetUseCase)(nil)

func NewResetUseCase(hub domain.Hub, lockManager lock.LockManager) ResetUseCase {
	return ResetUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc ResetUseCase) Execute(ctx context.Context, cmd ResetCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, ok := uc.hub.GetRoom(ctx, cmd.RoomID)
		if !ok {
			return fmt.Errorf("room %s not found", cmd.RoomID)
		}

		if err := room.ResetVoting(ctx, cmd.SenderID); err != nil {
			return err
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return err
		}

		return nil
	})
}
