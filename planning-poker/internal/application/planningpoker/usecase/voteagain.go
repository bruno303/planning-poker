package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	VoteAgainCommand struct {
		RoomID   string
		SenderID string
	}
	VoteAgainUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
	}
)

func NewVoteAgainUseCase(hub domain.Hub, lockManager lock.LockManager) VoteAgainUseCase {
	return VoteAgainUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc VoteAgainUseCase) Execute(ctx context.Context, cmd VoteAgainCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, ok := uc.hub.GetRoom(cmd.RoomID)
		if !ok {
			return fmt.Errorf("room %s not found", cmd.RoomID)
		}

		if err := room.ResetVoting(ctx, cmd.SenderID); err != nil {
			return err
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return err
		}

		return nil
	})
}
