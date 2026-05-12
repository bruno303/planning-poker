package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/http/middleware"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/mock/gomock"
)

func TestKickClientAPI_Endpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware("test-api-key"))

	expected := "/admin/rooms/{roomID}/client/{clientID}/kick"
	if api.Endpoint() != expected {
		t.Errorf("Endpoint() = %v, want %v", api.Endpoint(), expected)
	}
}

func TestKickClientAPI_Methods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware("test-api-key"))

	expectedMethods := []string{"POST", "OPTIONS"}
	methods := api.Methods()
	if len(methods) != len(expectedMethods) {
		t.Fatalf("Methods() length = %v, want %v", len(methods), len(expectedMethods))
	}
	for i, method := range expectedMethods {
		if methods[i] != method {
			t.Errorf("Methods()[%d] = %v, want %v", i, methods[i], method)
		}
	}
}

func TestKickClientAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"
	roomID := "room123"
	clientID := "client456"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	mockUseCase.EXPECT().
		Execute(gomock.Any(), usecase.AdminKickClientCommand{RoomID: roomID, ClientID: clientID}).
		Return(nil)

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}/client/{clientID}/kick", handler).Methods("POST", "OPTIONS")

	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/room123/client/client456/kick", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "kicked" {
		t.Errorf("response status = %v, want %v", response["status"], "kicked")
	}
}

func TestKickClientAPI_Handle_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	mockUseCase.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("load room: %w", domain.ErrRoomNotFound))

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}/client/{clientID}/kick", handler).Methods("POST")

	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/nonexistent/client/client456/kick", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusNotFound)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Room not found" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Room not found")
	}
}

func TestKickClientAPI_Handle_ClientNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	mockUseCase.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("not found: %w", domain.ErrClientNotFound))

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}/client/{clientID}/kick", handler).Methods("POST")

	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/room123/client/nonexistent-client/kick", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusNotFound)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Client not found" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Client not found")
	}
}

func TestKickClientAPI_Handle_UseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	mockUseCase.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(errors.New("some internal error"))

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}/client/{clientID}/kick", handler).Methods("POST")

	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/room123/client/client456/kick", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusInternalServerError)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Failed to kick client" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Failed to kick client")
	}
}

func TestKickClientAPI_Handle_UnauthorizedWrongKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/room123/client/client456/kick", nil)
	req.Header.Set("Authorization", "Bearer wrong-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestKickClientAPI_Handle_UnauthorizedMissingHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/room123/client/client456/kick", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestKickClientAPI_Handle_MissingRoomID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/admin/rooms//client/client456/kick", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusBadRequest)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Room ID is required" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Room ID is required")
	}
}

func TestKickClientAPI_Handle_MissingClientID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	apiKey := "valid-api-key"

	mockUseCase := usecase.NewMockUseCase[usecase.AdminKickClientCommand](ctrl)
	api := NewKickClientAPI(mockUseCase, middleware.NewAdminMiddleware(apiKey))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodPost, "/admin/rooms/room123/client//kick", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	req = mux.SetURLVars(req, map[string]string{"roomID": "room123", "clientID": ""})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusBadRequest)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Client ID is required" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Client ID is required")
	}
}
