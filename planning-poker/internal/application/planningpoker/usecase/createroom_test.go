package usecase

import (
	"context"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewCreateRoomUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	uc := NewCreateRoomUseCase(mockHub, mockMetric)

	if uc.hub != mockHub {
		t.Error("hub not set correctly")
	}
	if uc.logger == nil {
		t.Error("logger not initialized")
	}
}

func TestCreateRoomUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	expectedRoom := &entity.Room{ID: "room123"}
	senderID := "user123"

	mockHub.EXPECT().
		NewRoom(ctx, senderID).
		Return(expectedRoom)

	uc := NewCreateRoomUseCase(mockHub, mockMetric)
	cmd := CreateRoomCommand{SenderID: senderID}

	output, err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Room != expectedRoom {
		t.Errorf("expected room %v, got %v", expectedRoom, output.Room)
	}
}

func TestCreateRoomUseCase_Execute_EmptySenderID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	mockMetric := metric.NewPlanningPokerMetric()

	expectedRoom := &entity.Room{ID: "room123"}

	mockHub.EXPECT().
		NewRoom(ctx, "").
		Return(expectedRoom)

	uc := NewCreateRoomUseCase(mockHub, mockMetric)
	cmd := CreateRoomCommand{SenderID: ""}

	output, err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Room != expectedRoom {
		t.Errorf("expected room %v, got %v", expectedRoom, output.Room)
	}
}
