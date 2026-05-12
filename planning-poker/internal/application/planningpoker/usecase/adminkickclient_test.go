package usecase

import (
	"context"
	"errors"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewAdminKickClientUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
	if uc.leaveRoom != mockLeaveRoom {
		t.Error("leaveRoom not set correctly")
	}
}

func TestAdminKickClientUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "client456"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: clientID}),
	}

	mockBus := domain.NewMockBus(ctrl)

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().GetBus(clientID).Return(mockBus, true)
	mockLeaveRoom.EXPECT().Execute(ctx, LeaveRoomCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
	mockBus.EXPECT().Send(ctx, dto.NewKickNotification()).Return(nil)
	mockBus.EXPECT().Close().Return(nil)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_Success_NoBus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "client456"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: clientID}),
	}

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().GetBus(clientID).Return(nil, false)
	mockLeaveRoom.EXPECT().Execute(ctx, LeaveRoomCommand{RoomID: roomID, SenderID: clientID}).Return(nil)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "nonexistent"
	clientID := "client456"

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(nil, domain.ErrRoomNotFound)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrRoomNotFound) {
		t.Errorf("expected ErrRoomNotFound, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_ClientNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "absentClient"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: "otherClient"}),
	}

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrClientNotFound) {
		t.Errorf("expected ErrClientNotFound, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_LeaveRoomError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "client456"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: clientID}),
	}

	leaveRoomErr := errors.New("leave room failed")

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().GetBus(clientID).Return(nil, false)
	mockLeaveRoom.EXPECT().Execute(ctx, LeaveRoomCommand{RoomID: roomID, SenderID: clientID}).Return(leaveRoomErr)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, leaveRoomErr) {
		t.Errorf("expected leaveRoom error, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_LeaveRoomError_WithBus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "client456"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: clientID}),
	}

	leaveRoomErr := errors.New("leave room failed")

	mockBus := domain.NewMockBus(ctrl)

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().GetBus(clientID).Return(mockBus, true)
	mockLeaveRoom.EXPECT().Execute(ctx, LeaveRoomCommand{RoomID: roomID, SenderID: clientID}).Return(leaveRoomErr)
	// No Send or Close expectations — they must NOT be called on leaveRoom failure

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, leaveRoomErr) {
		t.Errorf("expected leaveRoom error, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_BusCloseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "client456"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: clientID}),
	}

	mockBus := domain.NewMockBus(ctrl)

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().GetBus(clientID).Return(mockBus, true)
	mockLeaveRoom.EXPECT().Execute(ctx, LeaveRoomCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
	mockBus.EXPECT().Send(ctx, dto.NewKickNotification()).Return(nil)
	mockBus.EXPECT().Close().Return(errors.New("close error"))

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	// Bus.Close error is best effort, should still return nil
	if err != nil {
		t.Fatalf("expected no error despite bus close failure, got %v", err)
	}
}

func TestAdminKickClientUseCase_Execute_SendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockLeaveRoom := NewMockUseCase[LeaveRoomCommand](ctrl)

	roomID := "room123"
	clientID := "client456"
	room := &entity.Room{
		ID:      roomID,
		Clients: clientcollection.New(&entity.Client{ID: clientID}),
	}

	mockBus := domain.NewMockBus(ctrl)

	mockHub.EXPECT().LoadRoom(ctx, roomID).Return(room, nil)
	mockHub.EXPECT().GetBus(clientID).Return(mockBus, true)
	mockLeaveRoom.EXPECT().Execute(ctx, LeaveRoomCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
	mockBus.EXPECT().Send(ctx, dto.NewKickNotification()).Return(errors.New("send error"))
	mockBus.EXPECT().Close().Return(nil)

	uc := NewAdminKickClientUseCase(mockLeaveRoom, mockHub)
	err := uc.Execute(ctx, AdminKickClientCommand{RoomID: roomID, ClientID: clientID})

	// Send error is best effort, should still return nil
	if err != nil {
		t.Fatalf("expected no error despite send failure, got %v", err)
	}
}
