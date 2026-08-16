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
		room, autoCreated, err := uc.loadOrCreateRoom(ctx, cmd)
		if err != nil {
			return nil, err
		}

		client, isReconnect, rollbackFunc := uc.joinClient(ctx, room, cmd)

		output := &JoinRoomOutput{Client: client, Room: room}
		if err := uc.notifyJoin(ctx, cmd, room, client); err != nil {
			return output, uc.rollbackJoin(ctx, rollbackFunc, err)
		}

		uc.recordJoinMetrics(ctx, autoCreated, isReconnect)

		return output, nil
	})

	if err != nil {
		return nil, err
	}

	return output.(*JoinRoomOutput), nil
}

func (uc JoinRoomUseCase) loadOrCreateRoom(ctx context.Context, cmd JoinRoomCommand) (*entity.Room, bool, error) {
	room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
	if err == nil {
		return room, false, nil
	}
	if !errors.Is(err, domain.ErrRoomNotFound) {
		return nil, false, fmt.Errorf("failed to load room %s: %w", cmd.RoomID, err)
	}

	room, err = uc.hub.NewRoomWithID(ctx, cmd.RoomID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to auto-create room %s: %w", cmd.RoomID, err)
	}

	uc.logger.Info(ctx, "Room auto-created with ID: %s during join by: %s", room.ID, cmd.SenderID)
	return room, true, nil
}

func (uc JoinRoomUseCase) notifyJoin(ctx context.Context, cmd JoinRoomCommand, room *entity.Room, client *entity.Client) error {
	uc.logger.Debug(ctx, "sending update client ID command for client %s on room %s", client.ID, room.ID)
	if err := cmd.Bus.Send(ctx, dto.NewUpdateClientIDCommand(client.ID)); err != nil {
		return fmt.Errorf("failed to send update client ID command: %w", err)
	}

	uc.logger.Debug(ctx, "broadcasting room state for room %s", room.ID)
	if err := uc.hub.BroadcastToRoom(ctx, room.ID, dto.NewRoomStateCommand(room)); err != nil {
		return fmt.Errorf("failed to broadcast room state: %w", err)
	}

	return nil
}

func (uc JoinRoomUseCase) rollbackJoin(ctx context.Context, rollbackFunc func(context.Context) error, cause error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackJoinCleanupTimeout)
	defer cancel()

	if err := rollbackFunc(cleanupCtx); err != nil {
		return fmt.Errorf("%w: rollback join initialization: %w", cause, err)
	}

	return cause
}

func (uc JoinRoomUseCase) recordJoinMetrics(ctx context.Context, autoCreated, isReconnect bool) {
	if !isReconnect {
		uc.metric.IncrementUsersTotal(ctx)
		uc.metric.IncrementActiveUsers(ctx)
	}
	if autoCreated {
		uc.metric.IncrementActiveRoomsCounter(ctx)
	}
}

func (uc JoinRoomUseCase) joinClient(ctx context.Context, room *entity.Room, cmd JoinRoomCommand) (client *entity.Client, isReconnect bool, rollbackFunc func(context.Context) error) {
	if existingClient, ok := room.FindClient(cmd.SenderID); ok {
		isReconnect = true
		client = existingClient
		rollbackFunc = uc.reconnectClient(ctx, cmd)
	} else {
		client = room.NewClient(cmd.SenderID)
		rollbackFunc = uc.createNewClient(ctx, cmd, client)
	}

	return
}

func (uc JoinRoomUseCase) reconnectClient(ctx context.Context, cmd JoinRoomCommand) func(context.Context) error {
	uc.logger.Info(ctx, "Client %s reconnecting to room %s", cmd.SenderID, cmd.RoomID)

	if oldBus, ok := uc.hub.GetBus(cmd.SenderID); ok {
		oldBus.Detach()
		if err := oldBus.Close(); err != nil {
			uc.logger.Debug(ctx, "closing old bus for client %s: %v", cmd.SenderID, err)
		}
	}

	uc.hub.AddBus(ctx, cmd.SenderID, cmd.Bus)

	return func(cleanupCtx context.Context) error {
		uc.hub.RemoveBus(cleanupCtx, cmd.SenderID)
		return nil
	}
}

func (uc JoinRoomUseCase) createNewClient(ctx context.Context, cmd JoinRoomCommand, client *entity.Client) func(context.Context) error {
	uc.logger.Debug(ctx, "creating client for room %s", cmd.RoomID)
	uc.hub.AddClient(client)

	uc.logger.Debug(ctx, "creating bus for client %s on room %s", client.ID, cmd.RoomID)
	uc.hub.AddBus(ctx, client.ID, cmd.Bus)

	return func(cleanupCtx context.Context) error {
		return uc.hub.RemoveClient(cleanupCtx, client.ID, cmd.RoomID)
	}
}
