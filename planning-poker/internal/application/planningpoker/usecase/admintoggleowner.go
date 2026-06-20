package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	AdminToggleOwnerCommand struct {
		RoomID         string
		TargetClientID string
	}
	adminToggleOwnerUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
		logger      log.Logger
	}
)

var _ UseCase[AdminToggleOwnerCommand] = (*adminToggleOwnerUseCase)(nil)

func NewAdminToggleOwnerUseCase(hub domain.Hub, lockManager lock.LockManager) *adminToggleOwnerUseCase {
	return &adminToggleOwnerUseCase{
		hub:         hub,
		lockManager: lockManager,
		logger:      log.NewLogger("usecase.admintoggleowner"),
	}
}

func (uc *adminToggleOwnerUseCase) Execute(ctx context.Context, cmd AdminToggleOwnerCommand) error {
	uc.logger.Info(ctx, "Admin toggling owner for client %s in room %s", cmd.TargetClientID, cmd.RoomID)

	return uc.lockManager.ExecuteWithLock(ctx, cmd.RoomID, func(ctx context.Context) error {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		if err != nil {
			uc.logger.Error(ctx, "Error loading room", err)
			return fmt.Errorf("load room: %w", err)
		}

		if err := room.AdminToggleOwner(ctx, cmd.TargetClientID); err != nil {
			uc.logger.Warn(ctx, "Admin toggle owner failed: %v", err)
			return err
		}

		if err := uc.hub.SaveRoom(ctx, room); err != nil {
			uc.logger.Error(ctx, "Error saving room", err)
			return fmt.Errorf("save room: %w", err)
		}

		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			uc.logger.Error(ctx, "Error broadcasting room state", err)
			return err
		}

		uc.logger.Info(ctx, "Admin successfully toggled owner for client %s in room %s", cmd.TargetClientID, cmd.RoomID)
		return nil
	})
}
