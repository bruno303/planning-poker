package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"planning-poker/internal/application/planningpoker/usecase"

	"go.uber.org/mock/gomock"
)

func TestCreateRoomAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{Deck: "fibonacci"}).
		Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(`{"deck":"fibonacci"}`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
}

func TestCreateRoomAPI_Handle_WhenUseCaseFails_ReturnsInternalServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().Execute(gomock.Any(), usecase.CreateRoomCommand{Deck: "fibonacci"}).Return(usecase.CreateRoomOutput{}, errors.New("redis unavailable"))

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/planning/rooms", strings.NewReader(`{"deck":"fibonacci"}`))

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusInternalServerError, `{"error":"Failed to create room"}`)
}

func TestCreateRoomAPI_Handle_RejectsInvalidDeck(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	api := NewCreateRoomAPI(mockUseCase)
	for _, body := range []string{``, `{`, `{"deck":"custom"}`, `{}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(body))
		api.Handle().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want 400", body, recorder.Code)
		}
	}
}

func TestCreateRoomAPI_Handle_RejectsTrailingData(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseR[usecase.CreateRoomCommand, usecase.CreateRoomOutput](ctrl)
	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", strings.NewReader(`{"deck":"fibonacci"} garbage`))

	api.Handle().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
