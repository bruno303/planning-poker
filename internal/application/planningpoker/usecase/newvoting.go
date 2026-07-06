package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	NewVotingCommand struct {
		RoomID   string
		SenderID string
	}
	NewVotingUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[NewVotingCommand] = (*NewVotingUseCase)(nil)

func NewNewVotingUseCase(hub domain.Hub, lockManager lock.LockManager) NewVotingUseCase {
	return NewVotingUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc NewVotingUseCase) Execute(ctx context.Context, cmd NewVotingCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.NewVoting(ctx, cmd.SenderID); err != nil {
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
