package usecase

import (
	"context"
	"fmt"
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

func NewUpdateNameUseCase(hub domain.Hub) UpdateNameUseCase {
	return UpdateNameUseCase{
		hub: hub,
	}
}

func (uc UpdateNameUseCase) Execute(ctx context.Context, cmd UpdateNameCommand) error {
	room, ok := uc.hub.GetRoom(cmd.RoomID)
	if !ok {
		return fmt.Errorf("room %s not found", cmd.RoomID)
	}

	if err := room.UpdateClientName(ctx, cmd.SenderID, cmd.Username); err != nil {
		return err
	}

	if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
		return err
	}

	return nil
}
