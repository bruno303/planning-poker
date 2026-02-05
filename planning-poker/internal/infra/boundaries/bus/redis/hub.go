package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/clientcollection"
	"sync"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/redis/go-redis/v9"
)

const (
	roomKeyPrefix   = "planning-poker:room:"
	clientKeyPrefix = "planning-poker:client:"
	pubsubChannel   = "planning-poker:updates:"
	twentyFourHours = 24 * time.Hour
	cursorSize      = 100
)

type (
	RedisHub struct {
		client  *redis.Client
		logger  log.Logger
		buses   map[string]domain.Bus
		busMux  sync.RWMutex
		pubsub  *redis.PubSub
		closeCh chan struct{}
	}
	BroadcastMessage struct {
		RoomID  string `json:"roomId"`
		Payload any    `json:"payload"`
	}
)

var (
	_ domain.Hub      = (*RedisHub)(nil)
	_ domain.AdminHub = (*RedisHub)(nil)
)

func NewRedisHub(ctx context.Context, redisClient *redis.Client) (*RedisHub, error) {
	hub := &RedisHub{
		client:  redisClient,
		logger:  log.NewLogger("redis.hub"),
		buses:   make(map[string]domain.Bus),
		closeCh: make(chan struct{}),
	}
	hub.pubsub = redisClient.PSubscribe(ctx, pubsubChannel+"*")
	go hub.listenToPubSub(ctx)

	hub.logger.Info(ctx, "RedisHub initialized and listening to pub/sub")
	return hub, nil
}

func (h *RedisHub) Close() error {
	close(h.closeCh)
	if h.pubsub != nil {
		return h.pubsub.Close()
	}
	return nil
}

func (h *RedisHub) NewRoom(ctx context.Context, owner string) *entity.Room {
	room, _ := trace.Trace(ctx, trace.NameConfig("RedisHub", "NewRoom"), func(ctx context.Context) (any, error) {
		room := entity.NewRoom(clientcollection.New())
		if err := h.saveRoom(ctx, room); err != nil {
			h.logger.Error(ctx, "Failed to save new room to Redis", err)
			return nil, err
		}

		return room, nil
	})

	return room.(*entity.Room)
}

func (h *RedisHub) GetRoom(ctx context.Context, roomID string) (*entity.Room, bool) {
	room, err := h.loadRoom(ctx, roomID)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			h.logger.Error(ctx, fmt.Sprintf("Failed to load room %s from Redis", roomID), err)
		}
		return nil, false
	}
	return room, true
}

func (h *RedisHub) RemoveRoom(roomID string) {
	ctx := context.Background()
	key := roomKeyPrefix + roomID
	if err := h.client.Del(ctx, key).Err(); err != nil {
		h.logger.Error(ctx, fmt.Sprintf("Failed to delete room %s from Redis", roomID), err)
	}
}

func (h *RedisHub) FindClientByID(clientID string) (*entity.Client, bool) {
	ctx := context.Background()
	key := clientKeyPrefix + clientID

	roomID, err := h.client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			h.logger.Error(ctx, fmt.Sprintf("Failed to find client %s in Redis", clientID), err)
		}
		return nil, false
	}

	room, ok := h.GetRoom(ctx, roomID)
	if !ok {
		return nil, false
	}

	client, ok := room.Clients.Filter(func(c *entity.Client) bool {
		return c.ID == clientID
	}).First()

	return client, ok
}

func (h *RedisHub) AddClient(c *entity.Client) {
	ctx := context.Background()
	key := clientKeyPrefix + c.ID

	if err := h.client.Set(ctx, key, c.Room().ID, twentyFourHours).Err(); err != nil {
		h.logger.Error(ctx, fmt.Sprintf("Failed to save client %s to Redis", c.ID), err)
		return
	}

	// Save the updated room state to Redis after adding the client
	room := c.Room()
	if err := h.saveRoom(ctx, room); err != nil {
		h.logger.Error(ctx, fmt.Sprintf("Failed to save room %s after adding client %s", room.ID, c.ID), err)
	}
}

func (h *RedisHub) AddBus(clientID string, bus domain.Bus) {
	h.busMux.Lock()
	defer h.busMux.Unlock()
	h.buses[clientID] = bus
}

func (h *RedisHub) GetBus(clientID string) (domain.Bus, bool) {
	h.busMux.RLock()
	defer h.busMux.RUnlock()
	bus, ok := h.buses[clientID]
	return bus, ok
}

func (h *RedisHub) RemoveBus(clientID string) {
	h.busMux.Lock()
	defer h.busMux.Unlock()
	delete(h.buses, clientID)
}

func (h *RedisHub) RemoveClient(ctx context.Context, clientID string, roomID string) error {
	_, err := trace.Trace(ctx, trace.NameConfig("RedisHub", "RemoveClient"), func(ctx context.Context) (any, error) {
		clientKey := clientKeyPrefix + clientID
		if err := h.client.Del(ctx, clientKey).Err(); err != nil {
			h.logger.Error(ctx, fmt.Sprintf("Failed to delete client %s from Redis", clientID), err)
		}

		h.RemoveBus(clientID)

		room, ok := h.GetRoom(ctx, roomID)
		if !ok {
			return nil, fmt.Errorf("room %s not found", roomID)
		}

		if err := room.RemoveClient(ctx, clientID); err != nil {
			return nil, err
		}

		if room.IsEmpty() {
			h.RemoveRoom(room.ID)
		} else {
			if err := h.saveRoom(ctx, room); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

func (h *RedisHub) BroadcastToRoom(ctx context.Context, roomID string, message any) error {
	_, err := trace.Trace(ctx, trace.NameConfig("RedisHub", "BroadcastToRoom"), func(ctx context.Context) (any, error) {
		broadcastMsg := BroadcastMessage{
			RoomID:  roomID,
			Payload: message,
		}

		data, err := json.Marshal(broadcastMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal broadcast message: %w", err)
		}

		channel := pubsubChannel + roomID
		if err := h.client.Publish(ctx, channel, data).Err(); err != nil {
			return nil, fmt.Errorf("failed to publish message to Redis: %w", err)
		}

		return nil, nil
	})

	return err
}

func (h *RedisHub) GetRooms() []*entity.Room {
	ctx := context.Background()
	pattern := roomKeyPrefix + "*"

	var rooms []*entity.Room
	iter := h.client.Scan(ctx, cursorSize, pattern, 0).Iterator()
	for iter.Next(ctx) {
		roomKey := iter.Val()
		roomID := roomKey[len(roomKeyPrefix):]

		room, ok := h.GetRoom(ctx, roomID)
		if ok {
			rooms = append(rooms, room)
		}
	}

	if err := iter.Err(); err != nil {
		h.logger.Error(ctx, "Failed to scan rooms from Redis", err)
	}

	return rooms
}

func (h *RedisHub) SaveRoom(ctx context.Context, room *entity.Room) error {
	return h.saveRoom(ctx, room)
}

func (h *RedisHub) saveRoom(ctx context.Context, room *entity.Room) error {
	data, err := SerializeRoom(room)
	if err != nil {
		return fmt.Errorf("failed to serialize room: %w", err)
	}

	key := roomKeyPrefix + room.ID
	if err := h.client.Set(ctx, key, data, twentyFourHours).Err(); err != nil {
		return fmt.Errorf("failed to save room to Redis: %w", err)
	}

	return nil
}

func (h *RedisHub) loadRoom(ctx context.Context, roomID string) (*entity.Room, error) {
	key := roomKeyPrefix + roomID
	data, err := h.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	room, err := DeserializeRoom(data, clientcollection.New())
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize room: %w", err)
	}

	return room, nil
}

func (h *RedisHub) listenToPubSub(ctx context.Context) {
	ch := h.pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			if msg == nil {
				continue
			}

			var broadcastMsg BroadcastMessage
			if err := json.Unmarshal([]byte(msg.Payload), &broadcastMsg); err != nil {
				h.logger.Error(ctx, "Failed to unmarshal broadcast message", err)
				continue
			}

			h.forwardToLocalClients(ctx, broadcastMsg.RoomID, broadcastMsg.Payload)

		case <-h.closeCh:
			h.logger.Info(ctx, "Stopping pub/sub listener")
			return
		}
	}
}

func (h *RedisHub) forwardToLocalClients(ctx context.Context, roomID string, message any) {
	h.busMux.RLock()
	defer h.busMux.RUnlock()

	room, ok := h.GetRoom(ctx, roomID)
	if !ok {
		return
	}

	for _, client := range room.Clients.Values() {
		bus, ok := h.buses[client.ID]
		if !ok {
			continue
		}
		if err := bus.Send(ctx, message); err != nil {
			h.logger.Warn(ctx, fmt.Sprintf("Failed to send message to client %s: %v", client.ID, err))
		}
	}
}
