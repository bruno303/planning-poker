package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain/entity"

	"go.uber.org/mock/gomock"
)

func TestCreateRoomAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{}).
		Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", nil)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
}

func TestCreateRoomAPI_Handle_WhenUseCaseFails_ReturnsInternalServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().Execute(gomock.Any(), usecase.CreateRoomCommand{}).Return(usecase.CreateRoomOutput{}, errors.New("redis unavailable"))

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/planning/rooms", nil)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusInternalServerError, `{"error":"Failed to create room"}`)
}

func TestCreateRoomAPI_Handle_AcceptsTShirtDeck(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().Execute(gomock.Any(), usecase.CreateRoomCommand{DeckID: entity.DeckIDTShirt}).Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(`{"deckId":"tshirt"}`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
}

func TestCreateRoomAPI_Handle_DefaultsMissingOrEmptyDeck(t *testing.T) {
	for _, body := range []string{`{}`, `{"deckId":""}`} {
		t.Run(body, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
			mockUseCase.EXPECT().Execute(gomock.Any(), usecase.CreateRoomCommand{}).Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

			api := NewCreateRoomAPI(mockUseCase)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(body))
			api.Handle().ServeHTTP(recorder, request)

			assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
		})
	}
}

func TestCreateRoomAPI_Handle_RejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "wrong deck type", body: `{"deckId":42}`},
		{name: "null deck", body: `{"deckId":null}`},
		{name: "unknown deck", body: `{"deckId":"custom"}`},
		{name: "malformed JSON", body: `{"deckId":`},
		{name: "trailing JSON", body: `{} {}`},
		{name: "non-object body", body: `[]`},
		{name: "null body", body: `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
			if tt.name == "unknown deck" {
				mockUseCase.EXPECT().Execute(gomock.Any(), usecase.CreateRoomCommand{DeckID: "custom"}).Return(usecase.CreateRoomOutput{}, entity.ErrUnknownDeckID)
			}

			api := NewCreateRoomAPI(mockUseCase)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(tt.body))

			api.Handle().ServeHTTP(recorder, request)

			message := "Invalid room creation request"
			if tt.name == "unknown deck" {
				message = "Invalid deck ID"
			}
			assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"`+message+`"}`)
		})
	}
}
