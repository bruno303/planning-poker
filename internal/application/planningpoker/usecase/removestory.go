package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	RemoveStoryCommand struct {
		RoomID     string
		SenderID   string
		StoryIndex int
	}
	RemoveStoryUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[RemoveStoryCommand] = (*RemoveStoryUseCase)(nil)

func NewRemoveStoryUseCase(hub domain.Hub, lockManager lock.LockManager) RemoveStoryUseCase {
	return RemoveStoryUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc RemoveStoryUseCase) Execute(ctx context.Context, cmd RemoveStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.RemoveStory(ctx, cmd.SenderID, cmd.StoryIndex); err != nil {
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
