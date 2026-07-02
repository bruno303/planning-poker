package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	PrevStoryCommand struct {
		RoomID   string
		SenderID string
	}
	PrevStoryUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[PrevStoryCommand] = (*PrevStoryUseCase)(nil)

func NewPrevStoryUseCase(hub domain.Hub, lockManager lock.LockManager) PrevStoryUseCase {
	return PrevStoryUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc PrevStoryUseCase) Execute(ctx context.Context, cmd PrevStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.PrevStory(ctx, cmd.SenderID); err != nil {
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
