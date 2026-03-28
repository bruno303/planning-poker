package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	VoteAgainCommand struct {
		RoomID   string
		SenderID string
	}
	voteAgainUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ UseCase[VoteAgainCommand] = (*voteAgainUseCase)(nil)

func NewVoteAgainUseCase(hub domain.Hub, lockManager lock.LockManager) *voteAgainUseCase {
	return &voteAgainUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc *voteAgainUseCase) Execute(ctx context.Context, cmd VoteAgainCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		if err := room.ResetVoting(ctx, cmd.SenderID); err != nil {
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
