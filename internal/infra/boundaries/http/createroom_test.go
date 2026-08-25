package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"planning-poker/internal/application/planningpoker/usecase"

	"go.uber.org/mock/gomock"
)

func TestCreateRoomAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseO[usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().
		Execute(gomock.Any()).
		Return(usecase.CreateRoomOutput{RoomID: "room-123"}, nil)

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/planning/rooms", nil)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusCreated, `{"roomId":"room-123"}`)
}

func TestCreateRoomAPI_Handle_WhenUseCaseFails_ReturnsInternalServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUseCase := usecase.NewMockUseCaseO[usecase.CreateRoomOutput](ctrl)
	mockUseCase.EXPECT().Execute(gomock.Any()).Return(usecase.CreateRoomOutput{}, errors.New("redis unavailable"))

	api := NewCreateRoomAPI(mockUseCase)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/planning/rooms", nil)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusInternalServerError, `{"error":"Failed to create room"}`)
}
