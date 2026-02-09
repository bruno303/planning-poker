package redis

import (
	"context"
	"testing"
	"time"

	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/clientcollection"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRedisHub_NewRoom_SaveRoom_GetRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	room := entity.NewRoom(clientcollection.New())
	room.ID = "room1"

	// Serialize room for mock return
	roomBytes, _ := SerializeRoom(room)
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(roomBytes))

	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:room:room1", gomock.Any(), time.Duration(24*time.Hour)).Return(statusCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room1").Return(stringCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	// Set up expectations for Get before SaveRoom and GetRoom
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room1").Return(stringCmd).AnyTimes()

	// SaveRoom
	err := hub.SaveRoom(context.Background(), room)
	assert.NoError(t, err)

	// GetRoom
	gotRoom, ok := hub.GetRoom(context.Background(), "room1")
	assert.True(t, ok)
	assert.Equal(t, room.ID, gotRoom.ID)
}

func TestRedisHub_AddClient_RemoveRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	room := entity.NewRoom(clientcollection.New())
	room.ID = "room2"
	client := room.NewClient("client1")
	client.WithRoom(room)
	room.Clients.Add(client)

	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:client:client1", room.ID, time.Duration(24*time.Hour)).Return(redis.NewStatusCmd(context.Background()))
	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:room:room2", gomock.Any(), time.Duration(24*time.Hour)).Return(redis.NewStatusCmd(context.Background()))

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	hub.AddClient(client)

	mockRedis.EXPECT().Del(gomock.Any(), "planning-poker:room:room2").Return(redis.NewIntCmd(context.Background()))
	hub.RemoveRoom(room.ID)
}

func TestRedisHub_FindClientByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	room := entity.NewRoom(clientcollection.New())
	room.ID = "room3"
	client := room.NewClient("client2")
	client.WithRoom(room)
	room.Clients.Add(client)

	// Serialize room for mock return
	roomBytes, _ := SerializeRoom(room)
	stringCmdRoom := redis.NewStringCmd(context.Background())
	stringCmdRoom.SetVal(string(roomBytes))
	stringCmdClient := redis.NewStringCmd(context.Background())
	stringCmdClient.SetVal(room.ID)

	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:client:client2").Return(stringCmdClient)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:"+room.ID).Return(stringCmdRoom)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	found, ok := hub.FindClientByID("client2")
	assert.True(t, ok)
	assert.Equal(t, client.ID, found.ID)
}

func TestRedisHub_RemoveClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	room := entity.NewRoom(clientcollection.New())
	room.ID = "room4"
	client := room.NewClient("client3")
	client.WithRoom(room)
	room.Clients.Add(client)

	// Serialize room for mock return
	roomBytes, _ := SerializeRoom(room)
	stringCmdRoom := redis.NewStringCmd(context.Background())
	stringCmdRoom.SetVal(string(roomBytes))
	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetVal(1)
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")

	mockRedis.EXPECT().Del(gomock.Any(), "planning-poker:client:client3").Return(intCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:"+room.ID).Return(stringCmdRoom)
	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:room:"+room.ID, gomock.Any(), time.Duration(24*time.Hour)).Return(statusCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	err := hub.RemoveClient(context.Background(), "client3", room.ID)
	assert.NoError(t, err)
}

func TestRedisHub_BroadcastToRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	roomID := "room5"
	mockRedis.EXPECT().Publish(gomock.Any(), "planning-poker:updates:"+roomID, gomock.Any()).Return(redis.NewIntCmd(context.Background()))

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	err := hub.BroadcastToRoom(context.Background(), roomID, map[string]string{"type": "test"})
	assert.NoError(t, err)
}

func TestRedisHub_GetRooms(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	mockRedis.EXPECT().Scan(gomock.Any(), uint64(100), "planning-poker:room:*", int64(0)).Return(redis.NewScanCmd(context.Background(), nil))

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	_ = hub.GetRooms()
}
