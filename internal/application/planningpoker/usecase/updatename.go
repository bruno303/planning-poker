package usecase

import (
	"context"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
)

type (
	UpdateNameCommand struct {
		RoomID   string
		SenderID string
		Username string
	}
	UpdateNameUseCase struct {
		hub domain.Hub
	}
)

var _ UseCase[UpdateNameCommand] = (*UpdateNameUseCase)(nil)

func NewUpdateNameUseCase(hub domain.Hub) UpdateNameUseCase {
	return UpdateNameUseCase{
		hub: hub,
	}
}

func (uc UpdateNameUseCase) Execute(ctx context.Context, cmd UpdateNameCommand) error {
	room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
	if err != nil {
		return err
	}

	if err := room.UpdateClientName(ctx, cmd.SenderID, cmd.Username); err != nil {
		return err
	}

	if err := uc.hub.SaveRoom(ctx, room); err != nil {
		return err
	}

	if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
		return err
	}

	return nil
}
