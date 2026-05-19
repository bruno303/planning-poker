package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/http/middleware"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/mock/gomock"
)

func TestGetRoomStateAPI_Endpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("test-api-key"))

	if api.Endpoint() != "/admin/rooms/{roomID}" {
		t.Errorf("Endpoint() = %v, want %v", api.Endpoint(), "/admin/rooms/{roomID}")
	}
}

func TestGetRoomStateAPI_Methods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("test-api-key"))
	methods := api.Methods()

	if len(methods) != 1 {
		t.Fatalf("Methods() length = %v, want %v", len(methods), 1)
	}
	if methods[0] != "GET" {
		t.Errorf("Methods()[0] = %v, want %v", methods[0], "GET")
	}
}

func TestGetRoomStateAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	apiKey := "valid-api-key"
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware(apiKey))

	client1 := &entity.Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
		IsOwner:     true,
	}
	client2 := &entity.Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: true,
		IsOwner:     false,
	}

	roomID := "room1"
	mockHub.EXPECT().
		LoadRoom(gomock.Any(), roomID).
		Return(&entity.Room{ID: roomID, Clients: clientcollection.New(client1, client2)}, nil)

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/room1", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	var response GetRoomStateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != roomID {
		t.Errorf("response.ID = %v, want %v", response.ID, roomID)
	}
	if len(response.Clients) != 2 {
		t.Fatalf("response.Clients length = %v, want %v", len(response.Clients), 2)
	}
	if response.Clients[0].ID != "client1" || response.Clients[0].Name != "Alice" || response.Clients[0].IsSpectator || !response.Clients[0].IsOwner {
		t.Errorf("response.Clients[0] = %+v, want client1/Alice/false/true", response.Clients[0])
	}
	if response.Clients[1].ID != "client2" || response.Clients[1].Name != "Bob" || !response.Clients[1].IsSpectator || response.Clients[1].IsOwner {
		t.Errorf("response.Clients[1] = %+v, want client2/Bob/true/false", response.Clients[1])
	}
}

func TestGetRoomStateAPI_Handle_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("valid-api-key"))

	roomID := "nonexistent"
	mockHub.EXPECT().
		LoadRoom(gomock.Any(), roomID).
		Return(nil, domain.ErrRoomNotFound)

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status code = %v, want %v", rec.Code, http.StatusNotFound)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Room not found" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Room not found")
	}
}

func TestGetRoomStateAPI_Handle_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("valid-api-key"))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/room123", nil)
	req.Header.Set("Authorization", "Bearer wrong-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetRoomStateAPI_Handle_MissingAuthHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("valid-api-key"))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/room123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetRoomStateAPI_Handle_MissingRoomID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("valid-api-key"))

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status code = %v, want %v", rec.Code, http.StatusBadRequest)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Room ID is required" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Room ID is required")
	}
}

func TestGetRoomStateAPI_Handle_LoadRoomError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("valid-api-key"))

	mockHub.EXPECT().
		LoadRoom(gomock.Any(), "room123").
		Return(nil, context.DeadlineExceeded)

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/room123", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %v, want %v", rec.Code, http.StatusInternalServerError)
	}

	var errorResponse map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResponse["error"] != "Failed to load room" {
		t.Errorf("error message = %v, want %v", errorResponse["error"], "Failed to load room")
	}
}

func TestGetRoomStateAPI_Handle_ContextPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockHub(ctrl)
	api := NewGetRoomStateAPI(mockHub, middleware.NewAdminMiddleware("valid-api-key"))

	var capturedCtx context.Context
	mockHub.EXPECT().
		LoadRoom(gomock.Any(), "room123").
		DoAndReturn(func(ctx context.Context, roomID string) (*entity.Room, error) {
			capturedCtx = ctx
			return &entity.Room{ID: roomID, Clients: clientcollection.New()}, nil
		})

	handler := api.Handle()
	router := mux.NewRouter()
	router.Handle("/admin/rooms/{roomID}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/room123", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if capturedCtx == nil {
		t.Error("context was not propagated to hub")
	}
}
