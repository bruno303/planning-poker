package redis

import (
	"context"
	"errors"
	"testing"

	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newErrorTestHub(mockRedis *MockRedisClient) *RedisHub {
	return &RedisHub{
		client:           mockRedis,
		logger:           log.NewLogger("test"),
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}
}

func TestRedisHub_NewRoom_WhenSaveFailsReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	wantErr := errors.New("Redis unavailable")
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetErr(wantErr)
	mockRedis.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), twentyFourHours).Return(statusCmd)

	hub := newErrorTestHub(mockRedis)
	room, err := hub.NewRoom(context.Background())

	assert.Nil(t, room)
	assert.ErrorIs(t, err, wantErr)
}

func TestRedisHub_LoadRoom_MapsRedisNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(redis.Nil)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:missing").Return(stringCmd)

	hub := newErrorTestHub(mockRedis)
	room, err := hub.LoadRoom(context.Background(), "missing")

	assert.Nil(t, room)
	assert.ErrorIs(t, err, domain.ErrRoomNotFound)
}

func TestRedisHub_LoadRoom_WhenRedisFailsWrapsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	wantErr := errors.New("connection refused")
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetErr(wantErr)
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room-1").Return(stringCmd)

	hub := newErrorTestHub(mockRedis)
	room, err := hub.LoadRoom(context.Background(), "room-1")

	assert.Nil(t, room)
	assert.ErrorIs(t, err, wantErr)
	assert.EqualError(t, err, "load room room-1: connection refused")
}

func TestRedisHub_LoadRoom_WhenStoredDataIsMalformedReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	stringCmd := redis.NewStringCmd(context.Background())
	stringCmd.SetVal("not-json")
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:room:room-1").Return(stringCmd)

	hub := newErrorTestHub(mockRedis)
	room, err := hub.LoadRoom(context.Background(), "room-1")

	assert.Nil(t, room)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize room")
}

func TestRedisHub_SaveRoom_WhenRedisFailsWrapsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	wantErr := errors.New("write failed")
	evalCmd := redis.NewCmd(context.Background())
	evalCmd.SetErr(wantErr)
	mockRedis.EXPECT().Eval(gomock.Any(), gomock.Any(), []string{"planning-poker:room:room-1"}, gomock.Any(), gomock.Any(), gomock.Any()).Return(evalCmd)

	hub := newErrorTestHub(mockRedis)
	room := &entity.Room{ID: "room-1", Clients: clientcollection.New()}
	expectedVersion := room.RoomVersion
	err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion)

	assert.ErrorIs(t, err, wantErr)
	assert.EqualError(t, err, "failed to conditionally save room to Redis: write failed")
}

func TestRedisHub_BroadcastToRoom_WhenMessageCannotBeMarshaledReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	hub := newErrorTestHub(mockRedis)

	err := hub.BroadcastToRoom(context.Background(), "room-1", func() {})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal broadcast message")
}

func TestRedisHub_BroadcastToRoom_WhenPublishFailsWrapsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	wantErr := errors.New("publish failed")
	publishCmd := redis.NewIntCmd(context.Background())
	publishCmd.SetErr(wantErr)
	mockRedis.EXPECT().Publish(gomock.Any(), "planning-poker:updates:room-1", gomock.Any()).Return(publishCmd)

	hub := newErrorTestHub(mockRedis)
	err := hub.BroadcastToRoom(context.Background(), "room-1", map[string]string{"type": "room-state"})

	assert.ErrorIs(t, err, wantErr)
	assert.EqualError(t, err, "failed to publish message to Redis: publish failed")
}

func TestRedisHub_GetRooms_WhenRedisFailsReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	wantErr := errors.New("scan failed")
	keysCmd := redis.NewStringSliceCmd(context.Background())
	keysCmd.SetErr(wantErr)
	mockRedis.EXPECT().Keys(gomock.Any(), "planning-poker:room:*").Return(keysCmd)

	hub := newErrorTestHub(mockRedis)

	assert.Nil(t, hub.GetRooms())
}

func TestRedisHub_AddClient_WhenClientMappingFailsDoesNotSaveRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	wantErr := errors.New("mapping write failed")
	statusCmd := redis.NewStatusCmd(context.Background())
	statusCmd.SetErr(wantErr)
	mockRedis.EXPECT().Set(gomock.Any(), "planning-poker:client:client-1", "room-1", twentyFourHours).Return(statusCmd)

	room := &entity.Room{ID: "room-1", Clients: clientcollection.New()}
	client := room.NewClient("client-1")
	client.WithRoom(room)
	hub := newErrorTestHub(mockRedis)

	hub.AddClient(client)
}

func TestRedisHub_AddBus_WithEmptyRoomIDDoesNotSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	mockBus := domain.NewMockBus(ctrl)
	mockBus.EXPECT().RoomID().Return("")
	hub := newErrorTestHub(mockRedis)

	err := hub.AddBus(context.Background(), "client-1", mockBus)

	assert.NoError(t, err)
	got, ok := hub.GetBus("client-1")
	assert.True(t, ok)
	assert.Equal(t, mockBus, got)
}

func newFailingPubSub(t *testing.T, roomID string) *redis.PubSub {
	t.Helper()
	pubsubClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	pubsub := pubsubClient.Subscribe(context.Background(), pubsubChannel+roomID)
	t.Cleanup(func() {
		_ = pubsub.Close()
		_ = pubsubClient.Close()
	})
	return pubsub
}

func TestRedisHub_AddBus_SubscriptionFailureRestoresPreviousBusAndCounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	clientID := "client-move"
	oldRoomID := "room-old"
	newRoomID := "room-new"

	oldBus := domain.NewMockBus(ctrl)
	oldBus.EXPECT().RoomID().Return(oldRoomID).AnyTimes()
	newBus := domain.NewMockBus(ctrl)
	newBus.EXPECT().RoomID().Return(newRoomID)

	mockRedis.EXPECT().Subscribe(gomock.Any(), pubsubChannel+newRoomID).Return(newFailingPubSub(t, newRoomID))

	hub := newErrorTestHub(mockRedis)
	hub.buses[clientID] = oldBus
	hub.roomClientCounts[oldRoomID] = 1

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := hub.AddBus(ctx, clientID, newBus)

	assert.Error(t, err)
	got, ok := hub.GetBus(clientID)
	assert.True(t, ok)
	assert.Equal(t, oldBus, got)
	assert.Equal(t, 1, hub.GetClientsOfRoom(oldRoomID), "old room count must be restored, not leaked")
	assert.Zero(t, hub.GetClientsOfRoom(newRoomID), "failed call must not leak a count for the destination room")
}

func TestRedisHub_AddBus_SubscriptionFailureOnSameRoomReplacementKeepsCounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	clientID := "client-same-room"
	roomID := "room-same"

	oldBus := domain.NewMockBus(ctrl)
	oldBus.EXPECT().RoomID().Return(roomID).AnyTimes()
	newBus := domain.NewMockBus(ctrl)
	newBus.EXPECT().RoomID().Return(roomID)

	mockRedis.EXPECT().Subscribe(gomock.Any(), pubsubChannel+roomID).Return(newFailingPubSub(t, roomID))

	hub := newErrorTestHub(mockRedis)
	hub.buses[clientID] = oldBus
	hub.roomClientCounts[roomID] = 1

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := hub.AddBus(ctx, clientID, newBus)

	assert.Error(t, err)
	got, ok := hub.GetBus(clientID)
	assert.True(t, ok)
	assert.Equal(t, oldBus, got)
	assert.Equal(t, 1, hub.GetClientsOfRoom(roomID), "failed same-room replacement must not change the count")
}

func TestRedisHub_FindClientByID_WhenRedisFailsReturnsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRedis := NewMockRedisClient(ctrl)
	getCmd := redis.NewStringCmd(context.Background())
	getCmd.SetErr(errors.New("lookup failed"))
	mockRedis.EXPECT().Get(gomock.Any(), "planning-poker:client:client-1").Return(getCmd)
	hub := newErrorTestHub(mockRedis)

	client, ok := hub.FindClientByID("client-1")

	assert.Nil(t, client)
	assert.False(t, ok)
}

func TestRedisHub_RemoveBus_WhenClientDoesNotExistIsSafe(t *testing.T) {
	hub := &RedisHub{
		client:           nil,
		logger:           log.NewLogger("test"),
		buses:            make(map[string]domain.Bus),
		closeCh:          make(chan struct{}),
		roomClientCounts: make(map[string]int),
	}

	hub.RemoveBus(context.Background(), "missing")
	assert.Zero(t, hub.GetClientsOfRoom("room-1"))
}
