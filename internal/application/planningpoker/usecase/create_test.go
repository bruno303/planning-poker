package usecase

import (
	"context"
	"errors"
	"testing"

	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestCreateRoomUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	room := &entity.Room{ID: "room-123", Clients: clientcollection.New()}

	mockHub.EXPECT().NewRoom(ctx).Return(room, nil)

	uc := NewCreateRoomUseCase(mockHub)

	got, err := uc.Execute(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.RoomID != room.ID {
		t.Fatalf("expected room ID %q, got %q", room.ID, got.RoomID)
	}
}

func TestCreateRoomUseCase_Execute_WhenHubFails_ReturnsErrorWithoutMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	wantErr := errors.New("redis unavailable")

	mockHub.EXPECT().NewRoom(ctx).Return(nil, wantErr)

	uc := NewCreateRoomUseCase(mockHub)

	got, err := uc.Execute(ctx)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got != (CreateRoomOutput{}) {
		t.Fatalf("expected empty output, got %#v", got)
	}
}

func TestCreateClientUseCase_Execute_ReturnsUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	planningPokerMetric, recorder := newTestPlanningPokerMetric(ctrl)

	uc := NewCreateClientUseCase(domain.NewMockHub(ctrl), planningPokerMetric)

	got, err := uc.Execute(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ClientID == "" {
		t.Fatal("expected a client ID")
	}
	if _, err := uuid.Parse(got.ClientID); err != nil {
		t.Fatalf("expected UUID client ID, got %q: %v", got.ClientID, err)
	}
	if calls := recorder.getCalls(); len(calls) != 0 {
		t.Fatalf("expected create client not to update metrics, got %d calls", len(calls))
	}
}
