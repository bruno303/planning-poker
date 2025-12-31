package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain/entity"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestCreateRoomAPI_Endpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)

	if api.Endpoint() != "/planning/room" {
		t.Errorf("Endpoint() = %v, want %v", api.Endpoint(), "/planning/room")
	}
}

func TestCreateRoomAPI_Methods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)
	methods := api.Methods()

	expectedMethods := []string{"POST", "OPTIONS"}
	if len(methods) != len(expectedMethods) {
		t.Fatalf("Methods() length = %v, want %v", len(methods), len(expectedMethods))
	}

	for i, method := range expectedMethods {
		if methods[i] != method {
			t.Errorf("Methods()[%d] = %v, want %v", i, methods[i], method)
		}
	}
}

func TestCreateRoomAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)

	requestBody := CreateRoomRequest{
		CreatedBy: "user123",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	expectedRoom := &entity.Room{
		ID: "room123",
	}

	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{SenderID: "user123"}).
		Return(usecase.CreateRoomOutput{Room: expectedRoom}, nil)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/planning/room", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusCreated)
	}

	var response CreateRoomResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.RoomID != "room123" {
		t.Errorf("RoomID = %v, want %v", response.RoomID, "room123")
	}
}

func TestCreateRoomAPI_Handle_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/planning/room", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusBadRequest)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Invalid request body" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Invalid request body")
	}
}

func TestCreateRoomAPI_Handle_UseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)

	requestBody := CreateRoomRequest{
		CreatedBy: "user123",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	expectedError := errors.New("database error")
	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{SenderID: "user123"}).
		Return(usecase.CreateRoomOutput{}, expectedError)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/planning/room", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusInternalServerError)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Failed to create room" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Failed to create room")
	}
}

func TestCreateRoomAPI_Handle_EmptyBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)

	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.CreateRoomCommand{SenderID: ""}).
		Return(usecase.CreateRoomOutput{Room: &entity.Room{ID: "room123"}}, nil)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/planning/room", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusCreated)
	}
}

func TestCreateRoomAPI_Handle_ContextPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockCreateRoomUseCase(ctrl)
	api := NewCreateRoomAPI(mockUseCase)

	requestBody := CreateRoomRequest{
		CreatedBy: "user123",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	var capturedCtx context.Context
	mockUseCase.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd usecase.CreateRoomCommand) (usecase.CreateRoomOutput, error) {
			capturedCtx = ctx
			return usecase.CreateRoomOutput{Room: &entity.Room{ID: "room123"}}, nil
		})

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/planning/room", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedCtx == nil {
		t.Error("context was not propagated to use case")
	}
}
