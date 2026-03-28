package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRedisHub_NewRoom_SaveRoom_LoadRoom(t *testing.T) {
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

	// LoadRoom
	gotRoom, err := hub.LoadRoom(context.Background(), "room1")
	assert.NoError(t, err)
	assert.Equal(t, room.ID, gotRoom.ID)
}

func TestRedisHub_NewRoomWithID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")

	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:room:room-explicit", gomock.Any(), time.Duration(24*time.Hour)).Return(statusCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	room, err := hub.NewRoomWithID(context.Background(), "room-explicit")
	assert.NoError(t, err)
	assert.NotNil(t, room)
	assert.Equal(t, "room-explicit", room.ID)
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
	anotherClient := room.NewClient("client4")
	anotherClient.WithRoom(room)
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

func TestRedisHub_RemoveClient_MissingRoomStillCleansUpAndSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetVal(1)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.Nil)
	mockBus := domain.NewMockBus(ctrl)
	mockBus.EXPECT().RoomID().Return("room4").AnyTimes()

	mockRedis.EXPECT().Del(gomock.Any(), "planning-poker:client:client3").Return(intCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room4").Return(stringCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            map[string]domain.Bus{"client3": mockBus},
		closeCh:          make(chan struct{}),
		roomClientCounts: map[string]int{"room4": 1},
	}

	err := hub.RemoveClient(context.Background(), "client3", "room4")
	assert.NoError(t, err)
	_, ok := hub.GetBus("client3")
	assert.False(t, ok)
	assert.Zero(t, hub.GetClientsOfRoom("room4"))
}

func TestRedisHub_RemoveClient_ClientDeleteFailsAndRoomSavesSuccessfully_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	room := entity.NewRoom(clientcollection.New())
	room.ID = "room4"
	client := room.NewClient("client3")
	client.WithRoom(room)
	anotherClient := room.NewClient("client4")
	anotherClient.WithRoom(room)

	deleteErr := errors.New("delete failed")
	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetErr(deleteErr)
	roomBytes, _ := SerializeRoom(room)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(roomBytes))
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetVal("OK")
	mockBus := domain.NewMockBus(ctrl)
	mockBus.EXPECT().RoomID().Return("room4").AnyTimes()

	mockRedis.EXPECT().Del(gomock.Any(), "planning-poker:client:client3").Return(intCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room4").Return(stringCmd)
	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:room:room4", gomock.Any(), time.Duration(24*time.Hour)).Return(statusCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            map[string]domain.Bus{"client3": mockBus},
		closeCh:          make(chan struct{}),
		roomClientCounts: map[string]int{"room4": 1},
	}

	err := hub.RemoveClient(context.Background(), "client3", "room4")
	assert.NoError(t, err)
	_, ok := hub.GetBus("client3")
	assert.False(t, ok)
	assert.Zero(t, hub.GetClientsOfRoom("room4"))
}

func TestRedisHub_RemoveClient_SaveRoomFails_PropagatesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	room := entity.NewRoom(clientcollection.New())
	room.ID = "room4"
	client := room.NewClient("client3")
	client.WithRoom(room)
	anotherClient := room.NewClient("client4")
	anotherClient.WithRoom(room)

	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetVal(1)
	roomBytes, _ := SerializeRoom(room)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal(string(roomBytes))
	saveErr := errors.New("save failed")
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetErr(saveErr)
	mockBus := domain.NewMockBus(ctrl)
	mockBus.EXPECT().RoomID().Return("room4").AnyTimes()

	mockRedis.EXPECT().Del(gomock.Any(), "planning-poker:client:client3").Return(intCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room4").Return(stringCmd)
	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:room:room4", gomock.Any(), time.Duration(24*time.Hour)).Return(statusCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            map[string]domain.Bus{"client3": mockBus},
		closeCh:          make(chan struct{}),
		roomClientCounts: map[string]int{"room4": 1},
	}

	err := hub.RemoveClient(context.Background(), "client3", "room4")
	assert.ErrorIs(t, err, saveErr)
	assert.EqualError(t, err, "failed to save room to Redis: save failed")
	_, ok := hub.GetBus("client3")
	assert.False(t, ok)
	assert.Zero(t, hub.GetClientsOfRoom("room4"))
}

func TestRedisHub_RemoveClient_LoadFailureStillCleansUpAndPropagatesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	logger := log.NewLogger("test")

	intCmd := redis.NewIntCmd(context.Background())
	intCmd.SetVal(1)
	loadErr := errors.New("redis unavailable")
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(loadErr)
	mockBus := domain.NewMockBus(ctrl)
	mockBus.EXPECT().RoomID().Return("room4").AnyTimes()

	mockRedis.EXPECT().Del(gomock.Any(), "planning-poker:client:client3").Return(intCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room4").Return(stringCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            map[string]domain.Bus{"client3": mockBus},
		closeCh:          make(chan struct{}),
		roomClientCounts: map[string]int{"room4": 1},
	}

	err := hub.RemoveClient(context.Background(), "client3", "room4")
	assert.ErrorIs(t, err, loadErr)
	assert.EqualError(t, err, "load room room4: redis unavailable")
	_, ok := hub.GetBus("client3")
	assert.False(t, ok)
	assert.Zero(t, hub.GetClientsOfRoom("room4"))
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

	validRoom := entity.NewRoom(clientcollection.New())
	validRoom.ID = "room-valid"
	validRoomBytes, _ := SerializeRoom(validRoom)

	scanCmd := redis.NewScanCmd(context.Background(), nil)
	scanCmd.SetVal([]string{"planning-poker:room:room-broken", "planning-poker:room:room-valid"}, 0)

	brokenRoomCmd := redis.NewStringCmd(context.Background())
	brokenRoomCmd.SetErr(errors.New("redis unavailable"))
	validRoomCmd := redis.NewStringCmd(context.Background())
	validRoomCmd.SetVal(string(validRoomBytes))

	mockRedis.EXPECT().Scan(gomock.Any(), uint64(100), "planning-poker:room:*", int64(0)).Return(scanCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room-broken").Return(brokenRoomCmd)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room-valid").Return(validRoomCmd)

	hub := &RedisHub{
		client:           mockRedis,
		logger:           logger,
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	rooms := hub.GetRooms()
	assert.Len(t, rooms, 1)
	assert.Equal(t, validRoom.ID, rooms[0].ID)
}
