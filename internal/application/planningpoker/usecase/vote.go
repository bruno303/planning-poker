package usecase

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
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

var _ UseCase[VoteCommand] = (*voteUseCase)(nil)

func NewVoteUseCase(hub domain.Hub, lockManager lock.LockManager) *voteUseCase {
	return &voteUseCase{
		hub:         hub,
		lockManager: lockManager,
	}
}

func (uc *voteUseCase) Execute(ctx context.Context, cmd VoteCommand) error {
	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			return err
		}

		var vote *entity.Vote
		if cmd.Vote != nil {
			vote = entity.VotePtr(*cmd.Vote)
		}

		if err := room.Vote(ctx, entity.ClientID(cmd.SenderID), vote); err != nil {
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
