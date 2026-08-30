package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	AddStoryCommand struct {
		RoomID    string
		SenderID  string
		StoryName string
	}
	AddStoryUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[AddStoryCommand] = (*AddStoryUseCase)(nil)

func NewAddStoryUseCase(hub domain.Hub, lockManager lock.LockManager) AddStoryUseCase {
	return AddStoryUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc AddStoryUseCase) Execute(ctx context.Context, cmd AddStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.AddStory(ctx, cmd.SenderID, cmd.StoryName); err != nil {
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
