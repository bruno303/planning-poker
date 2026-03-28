package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	UpdateStoryCommand struct {
		RoomID   string
		SenderID string
		Story    string
	}
	UpdateStoryUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[UpdateStoryCommand] = (*UpdateStoryUseCase)(nil)

func NewUpdateStoryUseCase(hub domain.Hub, lockManager lock.LockManager) UpdateStoryUseCase {
	return UpdateStoryUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc UpdateStoryUseCase) Execute(ctx context.Context, cmd UpdateStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {

		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.SetCurrentStory(ctx, cmd.SenderID, cmd.Story); err != nil {
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
