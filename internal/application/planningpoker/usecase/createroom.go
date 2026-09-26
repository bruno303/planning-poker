package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/domainerror"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomCommand struct{ DeckID string }
	CreateRoomOutput  struct {
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
	deck := entity.DefaultDeck()
	if cmd.DeckID != "" {
		var ok bool
		deck, ok = entity.DeckByID(cmd.DeckID)
		if !ok {
			return CreateRoomOutput{}, fmt.Errorf("%s %q: %w", domainerror.ErrUnknownDeck, cmd.DeckID, domainerror.ErrUnknownDeck)
		}
	}
	room, err := uc.hub.NewRoom(ctx, deck)
	if err != nil {
		return CreateRoomOutput{}, err
	}

	uc.logger.Info(ctx, "Room created with ID: %s", room.ID)
	uc.metric.IncrementActiveRoomsCounter(ctx)

	return CreateRoomOutput{RoomID: room.ID}, nil
}
