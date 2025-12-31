package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewUpdateNameUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)

	uc := NewUpdateNameUseCase(mockHub)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
}

func TestUpdateNameUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)

	roomID := "room123"
	senderID := "client123"
	username := "Alice"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	room.NewClient(senderID)

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(nil)

	uc := NewUpdateNameUseCase(mockHub)
	cmd := UpdateNameCommand{
		RoomID:   roomID,
		SenderID: senderID,
		Username: username,
	}

	err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUpdateNameUseCase_Execute_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)

	roomID := "nonexistent"

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(nil, false)

	uc := NewUpdateNameUseCase(mockHub)
	cmd := UpdateNameCommand{
		RoomID:   roomID,
		SenderID: "client123",
		Username: "Alice",
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "room nonexistent not found" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}

func TestUpdateNameUseCase_Execute_BroadcastError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)

	roomID := "room123"
	senderID := "client123"
	room := &entity.Room{
		ID:      roomID,
		Clients: inmemory.NewInMemoryClientCollection(),
	}

	room.NewClient(senderID)
	expectedError := errors.New("broadcast failed")

	mockHub.EXPECT().GetRoom(ctx, roomID).Return(room, true)
	mockHub.EXPECT().BroadcastToRoom(ctx, roomID, gomock.Any()).Return(expectedError)

	uc := NewUpdateNameUseCase(mockHub)
	cmd := UpdateNameCommand{
		RoomID:   roomID,
		SenderID: senderID,
		Username: "Alice",
	}

	err := uc.Execute(ctx, cmd)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}
