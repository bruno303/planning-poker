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
		room, _, err := uc.loadOrCreateRoom(ctx, cmd)
		if err != nil {
			return nil, err
		}

		// A client holds exactly one bus per instance. Capture the local state
		// this instance had before the join mutates it: the destination's local
		// state decides whether it gains the active-rooms counter, and the room
		// the client came from decides whether it must release it. Reconnecting to
		// the bus's own room is not a room switch, so it never changes that count.
		previousBus, hadLocalBus := uc.hub.Bus(cmd.SenderID)
		previousRoomID := roomIDOf(previousBus)
		switchingRooms := previousRoomID != "" && previousRoomID != cmd.RoomID
		destinationHadLocalClients := uc.hub.GetClientsOfRoom(cmd.RoomID) > 0
		previousRoomHasOtherClients := switchingRooms && uc.hub.GetClientsOfRoom(previousRoomID) > 1

		client, isReconnect, rollbackFunc, err := uc.joinClient(ctx, room, cmd, previousBus)
		if err != nil {
			return nil, uc.rollbackJoin(ctx, rollbackFunc, err)
		}

		output := &JoinRoomOutput{Client: client, Room: room}
		if err := uc.notifyJoin(ctx, cmd, room, client); err != nil {
			return output, uc.rollbackJoin(ctx, rollbackFunc, err)
		}

		uc.recordJoinMetrics(ctx, joinMetrics{
			isReconnect:                 isReconnect,
			hasLocalBus:                 hadLocalBus,
			destinationHadLocalClients:  destinationHadLocalClients,
			previouslyActiveRoom:        previousRoomID,
			previousRoomHasOtherClients: previousRoomHasOtherClients,
			switchingRooms:              switchingRooms,
		})

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

// joinMetrics carries the presence facts a join needs to keep the process-local
// active counters aligned with what this instance actually tracks.
type joinMetrics struct {
	// isReconnect is true when the client was already a member of the room.
	isReconnect bool
	// hasLocalBus is true when the client already held a bus on this instance,
	// regardless of the room that bus belongs to.
	hasLocalBus bool
	// destinationHadLocalClients is true when the joined room already counted as
	// locally active before this join.
	destinationHadLocalClients bool
	// previouslyActiveRoom is the room this instance counted as active before the
	// join, if any.
	previouslyActiveRoom string
	// previousRoomHasOtherClients is true when previouslyActiveRoom keeps at least
	// one other local client after the join.
	previousRoomHasOtherClients bool
	// switchingRooms is true when the client moved from another room on this
	// instance. Reconnecting to the same room never changes the room count.
	switchingRooms bool
}

// activeRoomsDelta reports the net change the join causes to the count of rooms
// this instance considers locally active: the destination room may become
// active, and the room the client came from stops being active when the client
// was its only local client.
func (m joinMetrics) activeRoomsDelta() int {
	delta := 0
	if m.switchingRooms && !m.previousRoomHasOtherClients {
		delta--
	}
	if !m.destinationHadLocalClients {
		delta++
	}

	return delta
}

func (uc JoinRoomUseCase) recordJoinMetrics(ctx context.Context, m joinMetrics) {
	if !m.isReconnect {
		uc.metric.IncrementUsersTotal(ctx)
	}
	if !m.hasLocalBus {
		uc.metric.IncrementActiveUsers(ctx)
	}

	switch m.activeRoomsDelta() {
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
	// A client can still hold a bus for a different room on this instance when
	// the socket switch is mid-flight. AddBus replaces that map entry; the old
	// session's own close handler is what leaves its room, and reconnectClient
	// below is careful not to close a socket that belongs to another room.
	rollbackFunc, err = uc.createNewClient(ctx, cmd, client)

	return
}

// roomIDOf returns the room a bus belongs to, or an empty string when there is
// no bus.
func roomIDOf(bus domain.Bus) string {
	if bus == nil {
		return ""
	}

	return bus.RoomID()
}

func (uc JoinRoomUseCase) reconnectClient(ctx context.Context, cmd JoinRoomCommand, previousBus domain.Bus) (func(context.Context) error, error) {
	uc.logger.Info(ctx, "Client %s reconnecting to room %s", cmd.SenderID, cmd.RoomID)

	// Only close the previous socket when it belongs to the same room. A bus
	// registered for another room belongs to a different session and must be
	// left alone; AddBus below replaces the map entry regardless.
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
