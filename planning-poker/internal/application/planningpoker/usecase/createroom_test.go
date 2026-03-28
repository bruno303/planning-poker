package usecase

import (
	"context"
	"errors"
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
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	expectedRoom := &entity.Room{ID: "room123"}
	senderID := "user123"

	mockHub.EXPECT().
		NewRoom(ctx).
		Return(expectedRoom, nil)

	uc := NewCreateRoomUseCase(mockHub, testMetric)
	cmd := CreateRoomCommand{SenderID: senderID}

	output, err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Room != expectedRoom {
		t.Errorf("expected room %v, got %v", expectedRoom, output.Room)
	}

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls, expectedMetricCall{name: metric.PlanningPokerActiveRoomsMetric, value: 1})
}

func TestCreateRoomUseCase_Execute_WithExplicitRoomID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)

	expectedRoom := &entity.Room{ID: "room-123"}

	mockHub.EXPECT().
		NewRoomWithID(ctx, expectedRoom.ID).
		Return(expectedRoom, nil)

	uc := NewCreateRoomUseCase(mockHub, testMetric)
	cmd := CreateRoomCommand{SenderID: "user123", RoomID: expectedRoom.ID}

	output, err := uc.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Room != expectedRoom {
		t.Errorf("expected room %v, got %v", expectedRoom, output.Room)
	}

	calls := metricMeter.getCalls()
	assertMetricCallSequence(t, calls, expectedMetricCall{name: metric.PlanningPokerActiveRoomsMetric, value: 1})
}

func TestCreateRoomUseCase_Execute_RoomCreationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedError := errors.New("create room failed")

	tests := []struct {
		name   string
		roomID string
		setup  func(mockHub *domain.MockHub)
	}{
		{
			name: "generated room id",
			setup: func(mockHub *domain.MockHub) {
				mockHub.EXPECT().NewRoom(ctx).Return(nil, expectedError)
			},
		},
		{
			name:   "explicit room id",
			roomID: "room-123",
			setup: func(mockHub *domain.MockHub) {
				mockHub.EXPECT().NewRoomWithID(ctx, "room-123").Return(nil, expectedError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHub := domain.NewMockHub(ctrl)
			testMetric, metricMeter := newTestPlanningPokerMetric(ctrl)
			tt.setup(mockHub)

			uc := NewCreateRoomUseCase(mockHub, testMetric)
			output, err := uc.Execute(ctx, CreateRoomCommand{SenderID: "user123", RoomID: tt.roomID})

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, expectedError) {
				t.Fatalf("expected error to wrap %v, got %v", expectedError, err)
			}
			if output.Room != nil {
				t.Fatalf("expected nil room on creation error, got %#v", output.Room)
			}

			calls := metricMeter.getCalls()
			if len(calls) != 0 {
				t.Fatalf("expected no metric changes on creation error, got %d calls", len(calls))
			}
		})
	}
}
