package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomCommand struct {
		DeckType entity.DeckType
	}
	CreateRoomOutput struct {
		RoomID string
	}
	CreateRoomUseCase struct {
		hub    domain.Hub
		logger log.Logger
	}
)

var _ UseCaseR[CreateRoomCommand, CreateRoomOutput] = (*CreateRoomUseCase)(nil)

func NewCreateRoomUseCase(hub domain.Hub) CreateRoomUseCase {
	return CreateRoomUseCase{
		hub:    hub,
		logger: log.NewLogger("usecase.CreateRoom"),
	}
}

func (uc CreateRoomUseCase) Execute(ctx context.Context, cmd CreateRoomCommand) (CreateRoomOutput, error) {
	if _, ok := entity.DeckByType(cmd.DeckType); !ok {
		return CreateRoomOutput{}, fmt.Errorf("unknown deck type %q", cmd.DeckType)
	}

	room, err := uc.hub.NewRoomWithDeck(ctx, cmd.DeckType)
	if err != nil {
		return CreateRoomOutput{}, err
	}

	uc.logger.Info(ctx, "Room created with ID: %s", room.ID)

	return CreateRoomOutput{RoomID: room.ID}, nil
}
