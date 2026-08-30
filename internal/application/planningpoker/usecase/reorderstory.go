package usecase

import (
	"context"

	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	ReorderStoryCommand struct {
		RoomID      string
		SenderID    string
		StoryID     string
		TargetIndex int
	}
	ReorderStoryUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[ReorderStoryCommand] = (*ReorderStoryUseCase)(nil)

func NewReorderStoryUseCase(hub domain.Hub, lockManager lock.LockManager) ReorderStoryUseCase {
	return ReorderStoryUseCase{hub: hub, lockManager: lockManager}
}

func (uc ReorderStoryUseCase) Execute(ctx context.Context, cmd ReorderStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.ReorderStory(ctx, cmd.SenderID, cmd.StoryID, cmd.TargetIndex); err != nil {
			return err
		}
		if err := uc.hub.SaveRoom(ctx, room, room.ExpectedPersistedRoomVersion()); err != nil {
			return err
		}
		return uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room))
	})
}
