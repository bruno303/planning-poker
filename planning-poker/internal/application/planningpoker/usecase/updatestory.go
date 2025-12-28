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

var _ application.UseCase[UpdateStoryCommand] = (*UpdateStoryUseCase)(nil)

func NewUpdateStoryUseCase(hub domain.Hub, lockManager lock.LockManager) UpdateStoryUseCase {
	return UpdateStoryUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc UpdateStoryUseCase) Execute(ctx context.Context, cmd UpdateStoryCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {

		room, ok := uc.hub.GetRoom(ctx, cmd.RoomID)
		if !ok {
			return fmt.Errorf("room %s not found", cmd.RoomID)
		}

		if err := room.SetCurrentStory(ctx, cmd.SenderID, cmd.Story); err != nil {
			return err
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return err
		}

		return nil
	})
}
