package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
)

type (
	JoinRoomCommand struct {
		RoomID     string
		SenderID   string
		BusFactory func(clientID string) domain.Bus
	}
	JoinRoomOutput struct {
		Client *entity.Client
		Room   *entity.Room
		Bus    domain.Bus
	}
	JoinRoomUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
		logger      log.Logger
		metric      metric.PlanningPokerMetric
	}
)

var _ application.UseCaseR[JoinRoomCommand, *JoinRoomOutput] = (*JoinRoomUseCase)(nil)

func NewJoinRoomUseCase(hub domain.Hub, lockManager lock.LockManager, metric metric.PlanningPokerMetric) JoinRoomUseCase {
	return JoinRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
		logger:      log.NewLogger("usecase.joinroom"),
		metric:      metric,
	}
}

func (uc JoinRoomUseCase) Execute(ctx context.Context, cmd JoinRoomCommand) (*JoinRoomOutput, error) {
	output, err := uc.lockManager.WithLock(ctx, cmd.RoomID, func(ctx context.Context) (any, error) {

		room, ok := uc.hub.GetRoom(ctx, cmd.RoomID)
		if !ok {
			return nil, fmt.Errorf("room %s not found", cmd.RoomID)
		}

		uc.logger.Debug(ctx, "creating client for room %s", room.ID)
		clientID := uuid.NewString()
		client := room.NewClient(clientID)
		uc.hub.AddClient(client)

		uc.logger.Debug(ctx, "creating bus for client %s on room %s", clientID, room.ID)
		bus := cmd.BusFactory(client.ID)
		uc.hub.AddBus(client.ID, bus)

		uc.metric.IncrementUsersTotal(ctx)
		uc.metric.IncrementActiveUsers(ctx)

		output := &JoinRoomOutput{Client: client, Room: room, Bus: bus}

		uc.logger.Debug(ctx, "sending update client ID command for client %s on room %s", clientID, room.ID)
		if err := bus.Send(ctx, dto.NewUpdateClientIDCommand(client.ID)); err != nil {
			return output, fmt.Errorf("failed to send update client ID command: %w", err)
		}

		uc.logger.Debug(ctx, "broadcasting room state for room %s", room.ID)
		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return output, fmt.Errorf("failed to broadcast room state: %w", err)
		}

		return output, nil
	})

	if err != nil {
		return nil, err
	}

	return output.(*JoinRoomOutput), nil
}
