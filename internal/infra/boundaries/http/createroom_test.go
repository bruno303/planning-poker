package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain/domainerror"

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

func TestCreateRoomAPI_Handle_WithDeckPreset_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{DeckID: "tshirt"}).
		Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(`{"deckPreset":"tshirt"}`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
}

func TestCreateRoomAPI_Handle_WithUnknownDeckPreset_ReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{DeckID: "unknown"}).
		Return(usecase.CreateRoomOutput{}, fmt.Errorf("deck: %w", domainerror.ErrUnknownDeck))

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(`{"deckPreset":"unknown"}`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"Unknown deck preset"}`)
}

func TestCreateRoomAPI_Handle_WithMalformedJSON_ReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(`{"deckPreset":`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"Malformed request body"}`)
}

func TestCreateRoomAPI_Handle_WhenUseCaseFails_ReturnsInternalServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(usecase.CreateRoomOutput{}, errors.New("redis unavailable"))

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/planning/rooms", nil)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusInternalServerError, `{"error":"Failed to create room"}`)
}
