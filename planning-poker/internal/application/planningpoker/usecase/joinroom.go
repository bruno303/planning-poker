package usecase

import (
	"context"
	"errors"
	"fmt"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
)

const rollbackJoinCleanupTimeout = 5 * time.Second

type (
	JoinRoomCommand struct {
		RoomID   string
		SenderID string
		Bus      domain.Bus
	}
	JoinRoomOutput struct {
		Client *entity.Client
		Room   *entity.Room
	}
	JoinRoomUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
		logger      log.Logger
		metric      metric.PlanningPokerMetric
	}
)

var _ UseCaseR[JoinRoomCommand, *JoinRoomOutput] = (*JoinRoomUseCase)(nil)

func NewJoinRoomUseCase(hub domain.Hub, lockManager lock.LockManager, metric metric.PlanningPokerMetric) JoinRoomUseCase {
	if hub == nil {
		panic("hub cannot be nil")
	}
	if lockManager == nil {
		panic("lockManager cannot be nil")
	}

	return JoinRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
		logger:      log.NewLogger("usecase.joinroom"),
		metric:      metric,
	}
}

func (uc JoinRoomUseCase) Execute(ctx context.Context, cmd JoinRoomCommand) (*JoinRoomOutput, error) {
	output, err := uc.lockManager.WithLock(ctx, cmd.RoomID, func(ctx context.Context) (any, error) {
		room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
		autoCreated := false
		if err != nil {
			if !errors.Is(err, domain.ErrRoomNotFound) {
				return nil, fmt.Errorf("failed to load room %s: %w", cmd.RoomID, err)
			}

			room, err = uc.hub.NewRoomWithID(ctx, cmd.RoomID)
			if err != nil {
				return nil, fmt.Errorf("failed to auto-create room %s: %w", cmd.RoomID, err)
			}

			autoCreated = true
			uc.logger.Info(ctx, "Room auto-created with ID: %s during join by: %s", room.ID, cmd.SenderID)
		}

		uc.logger.Debug(ctx, "creating client for room %s", room.ID)
		client := room.NewClient(cmd.SenderID)
		uc.hub.AddClient(client)

		uc.logger.Debug(ctx, "creating bus for client %s on room %s", client.ID, room.ID)
		uc.hub.AddBus(ctx, client.ID, cmd.Bus)

		output := &JoinRoomOutput{Client: client, Room: room}
		rollbackJoin := func(cause error) error {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackJoinCleanupTimeout)
			defer cancel()

			cleanupErr := uc.hub.RemoveClient(cleanupCtx, client.ID, room.ID)
			if cleanupErr != nil {
				return fmt.Errorf("%w: rollback join initialization: %w", cause, cleanupErr)
			}

			return cause
		}

		uc.logger.Debug(ctx, "sending update client ID command for client %s on room %s", client.ID, room.ID)
		if err := cmd.Bus.Send(ctx, dto.NewUpdateClientIDCommand(client.ID)); err != nil {
			return output, rollbackJoin(fmt.Errorf("failed to send update client ID command: %w", err))
		}

		uc.logger.Debug(ctx, "broadcasting room state for room %s", room.ID)
		if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
			return output, rollbackJoin(fmt.Errorf("failed to broadcast room state: %w", err))
		}

		if autoCreated {
			uc.metric.IncrementActiveRoomsCounter(ctx)
		}
		uc.metric.IncrementUsersTotal(ctx)
		uc.metric.IncrementActiveUsers(ctx)

		return output, nil
	})

	if err != nil {
		return nil, err
	}

	return output.(*JoinRoomOutput), nil
}
