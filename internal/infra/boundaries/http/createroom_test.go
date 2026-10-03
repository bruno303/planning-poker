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

func newCreateRoomRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(body))
}

func TestCreateRoomAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{DeckType: entity.DeckTypeTShirt}).
		Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := newCreateRoomRequest(`{"deckType":"t-shirt"}`)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
}

func TestCreateRoomAPI_Handle_MissingDeckType_ReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := newCreateRoomRequest(`{}`)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"Deck type is required"}`)
}

func TestCreateRoomAPI_Handle_UnknownDeckType_ReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := newCreateRoomRequest(`{"deckType":"planning"}`)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"Unknown deck type"}`)
}

func TestCreateRoomAPI_Handle_InvalidBody_ReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := newCreateRoomRequest(`not-json`)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"Invalid request body"}`)
}

func TestCreateRoomAPI_Handle_WhenUseCaseFails_ReturnsInternalServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{DeckType: entity.DeckTypeFibonacci}).
		Return(usecase.CreateRoomOutput{}, errors.New("redis unavailable"))

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/planning/rooms", strings.NewReader(`{"deckType":"fibonacci"}`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusInternalServerError, `{"error":"Failed to create room"}`)
}
