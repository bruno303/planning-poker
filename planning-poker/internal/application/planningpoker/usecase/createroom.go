package usecase

import (
	"context"
	"planning-poker/internal/application"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	CreateRoomCommand struct {
		SenderID string
	}
	CreateRoomOutput struct {
		Room *entity.Room
	}
	CreateRoomUseCase interface {
		Execute(ctx context.Context, cmd CreateRoomCommand) (CreateRoomOutput, error)
	}
	createRoomUseCase struct {
		hub    domain.Hub
		logger log.Logger
		metric metric.PlanningPokerMetric
	}
)

var _ application.UseCaseR[CreateRoomCommand, CreateRoomOutput] = (*createRoomUseCase)(nil)

func NewCreateRoomUseCase(hub domain.Hub, metric metric.PlanningPokerMetric) createRoomUseCase {
	return createRoomUseCase{
		hub:    hub,
		logger: log.NewLogger("usecase.CreateRoom"),
		metric: metric,
	}
}

func (uc createRoomUseCase) Execute(ctx context.Context, cmd CreateRoomCommand) (CreateRoomOutput, error) {
	room := uc.hub.NewRoom(ctx, cmd.SenderID)
	uc.metric.IncrementActiveRoomsCounter(ctx)

	uc.logger.Info(ctx, "Room created with ID: %s by: %s", room.ID, cmd.SenderID)

	return CreateRoomOutput{Room: room}, nil
}
