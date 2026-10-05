package inmemory

import (
	"context"
	"errors"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"sync"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/samber/lo"
)

type InMemoryHub struct {
	Rooms          map[string]*entity.Room
	Clients        map[string]*entity.Client
	Buses          map[string]domain.Bus
	logger         log.Logger
	roomMu         sync.Mutex
	busMu          sync.RWMutex
	saved          map[string]*entity.Room
	removed        map[string]chan struct{}
	removedClients map[string]chan struct{}
}

var _ domain.Hub = (*InMemoryHub)(nil)

func NewHub() *InMemoryHub {
	return &InMemoryHub{
		Rooms:          make(map[string]*entity.Room),
		Clients:        make(map[string]*entity.Client),
		Buses:          make(map[string]domain.Bus),
		saved:          make(map[string]*entity.Room),
		removed:        make(map[string]chan struct{}),
		removedClients: make(map[string]chan struct{}),
		logger:         log.NewLogger("inmemory.hub"),
	}
}

func (h *InMemoryHub) NewRoom(ctx context.Context) (*entity.Room, error) {
	room, _ := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "NewRoom"), func(ctx context.Context) (any, error) {
		room := entity.NewRoom(clientcollection.New())
		h.roomMu.Lock()
		h.Rooms[room.ID] = room
		h.saved[room.ID] = cloneRoom(room)
		h.removed[room.ID] = make(chan struct{})
		h.roomMu.Unlock()
		return room, nil
	})

	return room.(*entity.Room), nil
}

func (h *InMemoryHub) NewRoomWithID(ctx context.Context, roomID string) (*entity.Room, error) {
	room, _ := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "NewRoomWithID"), func(ctx context.Context) (any, error) {
		room := entity.NewRoomWithID(roomID, clientcollection.New())
		h.roomMu.Lock()
		h.Rooms[room.ID] = room
		h.saved[room.ID] = cloneRoom(room)
		h.removed[room.ID] = make(chan struct{})
		h.roomMu.Unlock()
		return room, nil
	})

	return room.(*entity.Room), nil
}

func (h *InMemoryHub) NewRoomWithDeck(ctx context.Context, deckType entity.DeckType) (*entity.Room, error) {
	room, _ := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "NewRoomWithDeck"), func(ctx context.Context) (any, error) {
		room := entity.NewRoomWithDeck(clientcollection.New(), deckType)
		h.roomMu.Lock()
		h.Rooms[room.ID] = room
		h.saved[room.ID] = cloneRoom(room)
		h.removed[room.ID] = make(chan struct{})
		h.roomMu.Unlock()
		return room, nil
	})

	return room.(*entity.Room), nil
}

func (h *InMemoryHub) LoadRoom(_ context.Context, roomID string) (*entity.Room, error) {
	room, ok := h.Rooms[roomID]
	if !ok {
		return nil, domain.ErrRoomNotFound
	}

	return room, nil
}

func (h *InMemoryHub) RemoveRoom(roomID string) {
	h.roomMu.Lock()
	defer h.roomMu.Unlock()
	delete(h.Rooms, roomID)
	delete(h.saved, roomID)
	if removed, ok := h.removed[roomID]; ok {
		close(removed)
		delete(h.removed, roomID)
	}
}

func (h *InMemoryHub) WaitForRoomRemoval(ctx context.Context, roomID string) error {
	h.roomMu.Lock()
	removed, ok := h.removed[roomID]
	if !ok {
		h.roomMu.Unlock()
		return nil
	}
	h.roomMu.Unlock()

	select {
	case <-removed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *InMemoryHub) WaitForClientRemoval(ctx context.Context, roomID, clientID string) error {
	h.roomMu.Lock()
	key := roomID + "\x00" + clientID
	removed, ok := h.removedClients[key]
	if !ok {
		h.roomMu.Unlock()
		return nil
	}
	h.roomMu.Unlock()

	select {
	case <-removed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *InMemoryHub) FindClientByID(clientID string) (*entity.Client, bool) {
	client, ok := h.Clients[clientID]
	return client, ok
}

func (h *InMemoryHub) AddClient(c *entity.Client) {
	h.roomMu.Lock()
	h.Clients[c.ID] = c
	room := c.Room()
	if room != nil {
		h.removedClients[room.ID+"\x00"+c.ID] = make(chan struct{})
	}
	h.roomMu.Unlock()
	if room != nil {
		h.refreshSavedRoom(room)
	}
}

func (h *InMemoryHub) AddBus(_ context.Context, clientID string, bus domain.Bus) error {
	h.busMu.Lock()
	defer h.busMu.Unlock()
	h.Buses[clientID] = bus
	return nil
}

func (h *InMemoryHub) GetClientsOfRoom(roomID string) int {
	h.busMu.RLock()
	defer h.busMu.RUnlock()
	count := 0
	for _, bus := range h.Buses {
		if bus.RoomID() == roomID {
			count++
		}
	}
	return count
}

func (h *InMemoryHub) BusIfInRoom(clientID string, roomID string) (domain.Bus, bool) {
	h.busMu.RLock()
	defer h.busMu.RUnlock()
	bus, ok := h.Buses[clientID]
	if !ok || bus.RoomID() != roomID {
		return nil, false
	}
	return bus, true
}

func (h *InMemoryHub) Bus(clientID string) (domain.Bus, bool) {
	h.busMu.RLock()
	defer h.busMu.RUnlock()
	bus, ok := h.Buses[clientID]
	return bus, ok
}

func (h *InMemoryHub) HasBusInRoom(clientID string, roomID string) bool {
	_, ok := h.BusIfInRoom(clientID, roomID)
	return ok
}

func (h *InMemoryHub) RemoveBus(_ context.Context, clientID string) {
	h.busMu.Lock()
	defer h.busMu.Unlock()
	delete(h.Buses, clientID)
}

// removeBusForRoom drops the client's bus only when it belongs to roomID. A
// client can hold a bus for another room on this instance (a room switch whose
// previous socket is still open), and leaving this room must not tear that
// connection down.
func (h *InMemoryHub) removeBusForRoom(clientID string, roomID string) {
	h.busMu.Lock()
	defer h.busMu.Unlock()
	bus, ok := h.Buses[clientID]
	if !ok || bus.RoomID() != roomID {
		return
	}
	delete(h.Buses, clientID)
}

func (h *InMemoryHub) RemoveClient(ctx context.Context, clientID string, roomID string) error {
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "RemoveClient"), func(ctx context.Context) (any, error) {

		delete(h.Clients, clientID)
		h.removeBusForRoom(clientID, roomID)

		room, err := h.LoadRoom(ctx, roomID)
		if err != nil {
			if errors.Is(err, domain.ErrRoomNotFound) {
				return nil, nil
			}

			return nil, err
		}

		err = room.RemoveClient(ctx, clientID)
		if err != nil {
			return nil, err
		}
		if room.IsEmpty() {
			h.RemoveRoom(room.ID)
		} else {
			h.refreshSavedRoom(room)
		}
		h.roomMu.Lock()
		key := roomID + "\x00" + clientID
		if removed, ok := h.removedClients[key]; ok {
			close(removed)
			delete(h.removedClients, key)
		}
		h.roomMu.Unlock()
		return nil, nil
	})

	return err
}

func (h *InMemoryHub) refreshSavedRoom(room *entity.Room) {
	h.roomMu.Lock()
	defer h.roomMu.Unlock()

	saved, ok := h.saved[room.ID]
	if !ok {
		return
	}
	snapshot := cloneRoom(room)
	snapshot.RoomVersion = saved.RoomVersion
	h.saved[room.ID] = snapshot
}

func (h *InMemoryHub) SaveRoom(_ context.Context, room *entity.Room) error {
	h.roomMu.Lock()
	defer h.roomMu.Unlock()
	next := room.RoomVersion + 1
	if saved, ok := h.saved[room.ID]; ok {
		next = saved.RoomVersion + 1
	}
	saved := cloneRoom(room)
	saved.RoomVersion = next
	h.saved[room.ID] = saved
	room.RoomVersion = next
	return nil
}

func (h *InMemoryHub) SaveRoomIfVersion(_ context.Context, room *entity.Room, expectedVersion *uint64) error {
	h.roomMu.Lock()
	defer h.roomMu.Unlock()

	if h.Rooms[room.ID] != room {
		return domain.ErrStaleRoomVersion
	}

	persistedVersion := uint64(0)
	if saved, ok := h.saved[room.ID]; ok {
		persistedVersion = saved.RoomVersion
	}
	if expectedVersion == nil || persistedVersion != *expectedVersion {
		if saved, ok := h.saved[room.ID]; ok && h.Rooms[room.ID] == room {
			h.Rooms[room.ID] = cloneRoom(saved)
		}
		return domain.ErrStaleRoomVersion
	}
	next := persistedVersion + 1
	saved := cloneRoom(room)
	saved.RoomVersion = next
	h.saved[room.ID] = saved
	room.RoomVersion = next
	h.Rooms[room.ID] = room
	return nil
}

func cloneRoom(room *entity.Room) *entity.Room {
	clone := entity.NewRoomWithIDAndStartedAt(room.ID, clientcollection.New(), room.StartedAt())
	clone.DeckType = room.DeckType
	clone.CurrentStory = room.CurrentStory
	clone.Reveal = room.Reveal
	clone.Result = cloneFloat32(room.Result)
	clone.MostAppearingVotes = append([]string(nil), room.MostAppearingVotes...)
	clone.Consensus = room.Consensus
	clone.LowestVote = cloneString(room.LowestVote)
	clone.HighestVote = cloneString(room.HighestVote)
	clone.VoteRange = cloneInt(room.VoteRange)
	clone.VoteSpread = cloneInt(room.VoteSpread)
	clone.SpecialVoteCount = room.SpecialVoteCount
	clone.BacklogMode = room.BacklogMode
	clone.Stories = append([]entity.Story(nil), room.Stories...)
	for i := range clone.Stories {
		clone.Stories[i].Result = cloneFloat32(room.Stories[i].Result)
		clone.Stories[i].MostAppearingVotes = append([]string(nil), room.Stories[i].MostAppearingVotes...)
	}
	clone.CurrentStoryIndex = room.CurrentStoryIndex
	clone.RoomVersion = room.RoomVersion
	if room.Clients != nil {
		for _, client := range room.Clients.Values() {
			copied := clone.NewClient(client.ID)
			copied.Name = client.Name
			copied.CurrentVote = cloneString(client.CurrentVote)
			copied.HasVoted = client.HasVoted
			copied.VotedAt = entity.OptionalUTCTimePtr(client.VotedAt)
			copied.IsSpectator = client.IsSpectator
			copied.IsOwner = client.IsOwner
		}
	}
	return clone
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneFloat32(value *float32) *float32 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (h *InMemoryHub) BroadcastToRoom(ctx context.Context, roomID string, message any) error {
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "BroadcastToRoom"), func(ctx context.Context) (any, error) {

		room, err := h.LoadRoom(ctx, roomID)
		if err != nil {
			return nil, err
		}

		for _, client := range room.Clients.Values() {
			bus, ok := h.BusIfInRoom(client.ID, roomID)
			if !ok {
				h.logger.Warn(ctx, "bus not found for client %s", client.ID)
				continue
			}
			if err := bus.Send(ctx, message); err != nil {
				return nil, fmt.Errorf("failed to send message to client %s: %w", client.ID, err)
			}
		}
		return nil, nil
	})

	return err
}

func (h *InMemoryHub) GetRooms() []*entity.Room {
	return lo.MapToSlice(h.Rooms, func(key string, room *entity.Room) *entity.Room {
		return room
	})
}
