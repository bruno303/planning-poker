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
	VoteCommand struct {
		RoomID   string
		SenderID string
		Vote     *string
	}
	voteUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

var _ application.UseCase[VoteCommand] = (*voteUseCase)(nil)

func NewVoteUseCase(hub domain.Hub, lockManager lock.LockManager) *voteUseCase {
	return &voteUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc *voteUseCase) Execute(ctx context.Context, cmd VoteCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, ok := uc.hub.GetRoom(ctx, cmd.RoomID)
		if !ok {
			return fmt.Errorf("room %s not found", cmd.RoomID)
		}

		if err := room.Vote(ctx, cmd.SenderID, cmd.Vote); err != nil {
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
