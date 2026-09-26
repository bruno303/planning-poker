package usecase

import (
	"context"
	"planning-poker/internal/domain"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomOutput struct {
		RoomID string
	}
	CreateRoomUseCase struct {
		hub    domain.Hub
		logger log.Logger
	}
)

var _ UseCaseO[CreateRoomOutput] = (*CreateRoomUseCase)(nil)

func NewCreateRoomUseCase(hub domain.Hub) CreateRoomUseCase {
	return CreateRoomUseCase{
		hub:    hub,
		logger: log.NewLogger("usecase.CreateRoom"),
	}
}

func (uc CreateRoomUseCase) Execute(ctx context.Context) (CreateRoomOutput, error) {
	room, err := uc.hub.NewRoom(ctx)
	if err != nil {
		return CreateRoomOutput{}, err
	}

	uc.logger.Info(ctx, "Room created with ID: %s", room.ID)

	return CreateRoomOutput{RoomID: room.ID}, nil
}
