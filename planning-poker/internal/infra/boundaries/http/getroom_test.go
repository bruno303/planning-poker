package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/mock/gomock"
)

func TestGetRoomAPI_Endpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomAPI(mockHub)

	if api.Endpoint() != "/planning/room/{roomID}" {
		t.Errorf("Endpoint() = %v, want %v", api.Endpoint(), "/planning/room/{roomID}")
	}
}

func TestGetRoomAPI_Methods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomAPI(mockHub)
	methods := api.Methods()

	if len(methods) != 1 {
		t.Fatalf("Methods() length = %v, want %v", len(methods), 1)
	}
	if methods[0] != "GET" {
		t.Errorf("Methods()[0] = %v, want %v", methods[0], "GET")
	}
}

func TestGetRoomAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomAPI(mockHub)

	roomID := "room123"
	expectedRoom := &entity.Room{
		ID: roomID,
	}

	mockHub.EXPECT().
		GetRoom(gomock.Any(), roomID).
		Return(expectedRoom, true)

	handler := api.Handle()

	// Use gorilla/mux router to properly set the route variables
	router := mux.NewRouter()
	router.Handle("/planning/room/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/planning/room/room123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	var response GetRoomResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.RoomID != roomID {
		t.Errorf("RoomID = %v, want %v", response.RoomID, roomID)
	}
}

func TestGetRoomAPI_Handle_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomAPI(mockHub)

	roomID := "nonexistent"

	mockHub.EXPECT().
		GetRoom(gomock.Any(), roomID).
		Return(nil, false)

	handler := api.Handle()

	router := mux.NewRouter()
	router.Handle("/planning/room/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/planning/room/nonexistent", nil)
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

func TestGetRoomAPI_Handle_MissingRoomID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomAPI(mockHub)

	handler := api.Handle()

	// Request without roomID in route variables
	req := httptest.NewRequest(http.MethodGet, "/planning/room/", nil)
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

func TestGetRoomAPI_Handle_ContextPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomAPI(mockHub)

	var capturedCtx context.Context
	mockHub.EXPECT().
		GetRoom(gomock.Any(), "room123").
		DoAndReturn(func(ctx context.Context, roomID string) (*entity.Room, bool) {
			capturedCtx = ctx
			return &entity.Room{ID: roomID}, true
		})

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/planning/room/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/planning/room/room123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if capturedCtx == nil {
		t.Error("context was not propagated to hub")
	}
}
