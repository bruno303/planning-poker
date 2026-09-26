package usecase

import (
	"context"
	"errors"
	"testing"

	"planning-poker/internal/domain"
	"planning-poker/internal/domain/domainerror"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestCreateRoomUseCase_Execute_UsesRequestedDeck(t *testing.T) {
	tshirtDeck, ok := entity.DeckByID("tshirt")
	if !ok {
		t.Fatal("tshirt deck preset not found")
	}

	tests := []struct {
		name   string
		deckID string
		want   entity.Deck
	}{
		{name: "defaults to fibonacci when deck is empty", deckID: "", want: entity.DefaultDeck()},
		{name: "uses the requested tshirt deck", deckID: "tshirt", want: tshirtDeck},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockHub := domain.NewMockHub(ctrl)
			planningPokerMetric, recorder := newTestPlanningPokerMetric(ctrl)
			room := &entity.Room{ID: "room-123", Clients: clientcollection.New()}

			mockHub.EXPECT().NewRoom(ctx, test.want).Return(room, nil)

			uc := NewCreateRoomUseCase(mockHub, planningPokerMetric)

			got, err := uc.Execute(ctx, CreateRoomCommand{DeckID: test.deckID})

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got.RoomID != room.ID {
				t.Fatalf("expected room ID %q, got %q", room.ID, got.RoomID)
			}
			assertMetricCallSequence(t, recorder.getCalls(), expectedMetricCall{
				name:  "planning_poker_active_rooms",
				value: 1,
			})
		})
	}
}

func TestCreateRoomUseCase_Execute_UnknownDeck_ReturnsErrorWithoutMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	planningPokerMetric, recorder := newTestPlanningPokerMetric(ctrl)

	uc := NewCreateRoomUseCase(mockHub, planningPokerMetric)

	got, err := uc.Execute(ctx, CreateRoomCommand{DeckID: "does-not-exist"})

	if !errors.Is(err, domainerror.ErrUnknownDeck) {
		t.Fatalf("expected %v, got %v", domainerror.ErrUnknownDeck, err)
	}
	if got != (CreateRoomOutput{}) {
		t.Fatalf("expected empty output, got %#v", got)
	}
	if calls := recorder.getCalls(); len(calls) != 0 {
		t.Fatalf("expected no metric calls, got %d", len(calls))
	}
}

func TestCreateRoomUseCase_Execute_WhenHubFails_ReturnsErrorWithoutMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	mockHub := domain.NewMockHub(ctrl)
	planningPokerMetric, recorder := newTestPlanningPokerMetric(ctrl)
	wantErr := errors.New("redis unavailable")

	mockHub.EXPECT().NewRoom(ctx, entity.DefaultDeck()).Return(nil, wantErr)

	uc := NewCreateRoomUseCase(mockHub, planningPokerMetric)

	got, err := uc.Execute(ctx, CreateRoomCommand{})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got != (CreateRoomOutput{}) {
		t.Fatalf("expected empty output, got %#v", got)
	}
	if calls := recorder.getCalls(); len(calls) != 0 {
		t.Fatalf("expected no metric calls, got %d", len(calls))
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
