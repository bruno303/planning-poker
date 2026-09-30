package usecase

import (
	"context"

	"github.com/bruno303/go-toolkit/pkg/log"

	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
)

type (
	CreateRoomCommand struct {
		DeckID entity.DeckID
	}

	CreateRoomOutput struct {
		RoomID string
	}
	CreateRoomUseCase struct {
		hub    domain.Hub
		logger log.Logger
		metric metric.PlanningPokerMetric
	}
)

var _ UseCaseR[CreateRoomCommand, CreateRoomOutput] = (*CreateRoomUseCase)(nil)

func NewCreateRoomUseCase(hub domain.Hub, metric metric.PlanningPokerMetric) CreateRoomUseCase {
	return CreateRoomUseCase{
		hub:    hub,
		logger: log.NewLogger("usecase.CreateRoom"),
		metric: metric,
	}
}

func (uc CreateRoomUseCase) Execute(ctx context.Context, cmd CreateRoomCommand) (CreateRoomOutput, error) {
	deck, err := entity.GetDeck(cmd.DeckID)
	if err != nil {
		return CreateRoomOutput{}, err
	}

	room, err := uc.hub.NewRoom(ctx, deck.ID)
	if err != nil {
		return CreateRoomOutput{}, err
	}

	uc.logger.Info(ctx, "Room created with ID: %s", room.ID)
	uc.metric.IncrementActiveRoomsCounter(ctx)

	return CreateRoomOutput{RoomID: room.ID}, nil
}
