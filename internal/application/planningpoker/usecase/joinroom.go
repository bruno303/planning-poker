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
		presence    PresenceGuard
		logger      log.Logger
		metric      metric.PlanningPokerMetric
	}
	joinState struct {
		client       *entity.Client
		isReconnect  bool
		hadLocalBus  bool
		previousBus  domain.Bus
		roomsBefore  int
		roomDelta    int
		rollbackFunc func(context.Context) error
	}
)

var _ UseCaseR[JoinRoomCommand, *JoinRoomOutput] = (*JoinRoomUseCase)(nil)

func NewJoinRoomUseCase(hub domain.Hub, lockManager lock.LockManager, metric metric.PlanningPokerMetric, presence PresenceGuard) JoinRoomUseCase {
	if hub == nil {
		panic("hub cannot be nil")
	}
	if lockManager == nil {
		panic("lockManager cannot be nil")
	}
	if presence == nil {
		panic("presence cannot be nil")
	}

	return JoinRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
		presence:    presence,
		logger:      log.NewLogger("usecase.joinroom"),
		metric:      metric,
	}
}

func (uc JoinRoomUseCase) Execute(ctx context.Context, cmd JoinRoomCommand) (*JoinRoomOutput, error) {
	output, err := uc.lockManager.WithLock(ctx, cmd.RoomID, func(ctx context.Context) (any, error) {
		room, _, err := uc.loadOrCreateRoom(ctx, cmd)
		if err != nil {
			return nil, err
		}

		state, err := uc.attachClient(ctx, room, cmd)
		if err != nil {
			return nil, err
		}

		output := &JoinRoomOutput{Client: state.client, Room: room}
		if err := uc.notifyJoin(ctx, cmd, room, state.client); err != nil {
			return output, uc.rollbackReplacement(ctx, cmd, state, err)
		}

		uc.recordJoinMetrics(ctx, state)

		return output, nil
	})

	if err != nil {
		return nil, err
	}

	return output.(*JoinRoomOutput), nil
}

// attachClient holds the presence guard only around the local presence
// bookkeeping. The room is already loaded and the guard is released before the
// room notification, so unrelated rooms are not blocked by socket or Redis I/O.
func (uc JoinRoomUseCase) attachClient(ctx context.Context, room *entity.Room, cmd JoinRoomCommand) (joinState, error) {
	unlock, err := uc.presence.Lock(ctx)
	if err != nil {
		return joinState{}, err
	}
	defer unlock()

	previousBus, hadLocalBus := uc.hub.GetBus(cmd.SenderID)
	previousRoomID := roomIDOf(previousBus)
	roomsBefore := activeRooms(uc.hub, cmd.RoomID, previousRoomID)

	state := joinState{
		hadLocalBus: hadLocalBus,
		previousBus: previousBus,
		roomsBefore: roomsBefore,
	}

	client, isReconnect, rollbackFunc, err := uc.joinClient(ctx, room, cmd, previousBus)
	state.client = client
	state.isReconnect = isReconnect
	state.rollbackFunc = rollbackFunc
	if err != nil {
		return state, uc.rollbackReplacementLocked(ctx, cmd, state, err)
	}
	state.roomDelta = activeRooms(uc.hub, cmd.RoomID, previousRoomID) - roomsBefore

	return state, nil
}

// activeRooms counts the given rooms that have at least one local client,
// ignoring empty ids and duplicates. It is the single helper used to derive
// active-room deltas on both the success and the rollback paths.
func activeRooms(hub domain.Hub, roomIDs ...string) int {
	seen := make(map[string]struct{}, len(roomIDs))
	count := 0
	for _, roomID := range roomIDs {
		if roomID == "" {
			continue
		}
		if _, ok := seen[roomID]; ok {
			continue
		}
		seen[roomID] = struct{}{}
		if hub.GetClientsOfRoom(roomID) > 0 {
			count++
		}
	}
	return count
}

func (uc JoinRoomUseCase) recordJoinMetrics(ctx context.Context, state joinState) {
	if !state.isReconnect {
		uc.metric.IncrementUsersTotal(ctx)
	}
	if !state.hadLocalBus {
		uc.metric.IncrementActiveUsers(ctx)
	}
	uc.recordRoomDelta(ctx, state.roomDelta)
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

func (uc JoinRoomUseCase) rollbackReplacement(ctx context.Context, cmd JoinRoomCommand, state joinState, cause error) error {
	guardCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackJoinCleanupTimeout)
	defer cancel()
	unlock, err := uc.presence.Lock(guardCtx)
	if err != nil {
		return fmt.Errorf("%w: acquire presence guard for rollback: %w", cause, err)
	}
	defer unlock()

	return uc.rollbackReplacementLocked(ctx, cmd, state, cause)
}

func (uc JoinRoomUseCase) rollbackReplacementLocked(ctx context.Context, cmd JoinRoomCommand, state joinState, cause error) error {
	err := uc.rollbackJoin(ctx, state.rollbackFunc, cause)
	if state.previousBus == nil {
		return err
	}

	previousRoomID := roomIDOf(state.previousBus)
	_, hasLocalBus := uc.hub.GetBus(cmd.SenderID)
	// A cross-room join leaves the original socket open. Restore it if cleanup
	// removed the replacement; a same-room reconnect already closed its socket.
	if !hasLocalBus && previousRoomID != cmd.RoomID {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackJoinCleanupTimeout)
		restoreErr := uc.hub.AddBus(cleanupCtx, cmd.SenderID, state.previousBus)
		cancel()
		if restoreErr != nil {
			err = fmt.Errorf("%w: restore previous bus: %w", err, restoreErr)
		}
		_, hasLocalBus = uc.hub.GetBus(cmd.SenderID)
	}
	if !hasLocalBus {
		uc.metric.DecrementActiveUsers(ctx)
	}

	roomsAfter := activeRooms(uc.hub, cmd.RoomID, previousRoomID)
	uc.recordRoomDelta(ctx, roomsAfter-state.roomsBefore)
	return err
}

func (uc JoinRoomUseCase) recordRoomDelta(ctx context.Context, delta int) {
	switch delta {
	case 1:
		uc.metric.IncrementActiveRoomsCounter(ctx)
	case -1:
		uc.metric.DecrementActiveRoomsCounter(ctx)
	}
}

func (uc JoinRoomUseCase) joinClient(ctx context.Context, room *entity.Room, cmd JoinRoomCommand, previousBus domain.Bus) (client *entity.Client, isReconnect bool, rollbackFunc func(context.Context) error, err error) {
	if existingClient, ok := room.FindClient(cmd.SenderID); ok {
		isReconnect = true
		client = existingClient
		rollbackFunc, err = uc.reconnectClient(ctx, cmd, previousBus)
		return
	}

	client = room.NewClient(cmd.SenderID)
	rollbackFunc, err = uc.createNewClient(ctx, cmd, client)

	return
}

func roomIDOf(bus domain.Bus) string {
	if bus == nil {
		return ""
	}

	return bus.RoomID()
}

func (uc JoinRoomUseCase) reconnectClient(ctx context.Context, cmd JoinRoomCommand, previousBus domain.Bus) (func(context.Context) error, error) {
	uc.logger.Info(ctx, "Client %s reconnecting to room %s", cmd.SenderID, cmd.RoomID)

	// Another room's socket remains usable if this join rolls back.
	if roomIDOf(previousBus) == cmd.RoomID {
		previousBus.Detach()
		if err := previousBus.Close(); err != nil {
			uc.logger.Debug(ctx, "closing old bus for client %s: %v", cmd.SenderID, err)
		}
	}

	rollbackFunc := func(cleanupCtx context.Context) error {
		uc.hub.RemoveBus(cleanupCtx, cmd.SenderID)
		return nil
	}
	if err := uc.hub.AddBus(ctx, cmd.SenderID, cmd.Bus); err != nil {
		if roomIDOf(previousBus) != cmd.RoomID {
			// AddBus restores the original bus when subscription setup fails.
			rollbackFunc = func(context.Context) error { return nil }
		}
		return rollbackFunc, fmt.Errorf("failed to add bus for client %s: %w", cmd.SenderID, err)
	}

	return rollbackFunc, nil
}

func (uc JoinRoomUseCase) createNewClient(ctx context.Context, cmd JoinRoomCommand, client *entity.Client) (func(context.Context) error, error) {
	uc.logger.Debug(ctx, "creating client for room %s", cmd.RoomID)
	uc.hub.AddClient(client)

	uc.logger.Debug(ctx, "creating bus for client %s on room %s", client.ID, cmd.RoomID)
	rollbackFunc := func(cleanupCtx context.Context) error {
		return uc.hub.RemoveClient(cleanupCtx, client.ID, cmd.RoomID)
	}
	if err := uc.hub.AddBus(ctx, client.ID, cmd.Bus); err != nil {
		return rollbackFunc, fmt.Errorf("failed to add bus for client %s: %w", client.ID, err)
	}

	return rollbackFunc, nil
}
