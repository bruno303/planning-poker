package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	AdvanceStoryCommand struct {
		RoomID   string
		SenderID string
	}
	AdvanceStoryUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[AdvanceStoryCommand] = (*AdvanceStoryUseCase)(nil)

func NewAdvanceStoryUseCase(hub domain.Hub, lockManager lock.LockManager) AdvanceStoryUseCase {
	return AdvanceStoryUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc AdvanceStoryUseCase) Execute(ctx context.Context, cmd AdvanceStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.AdvanceToNextStory(ctx, cmd.SenderID); err != nil {
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
