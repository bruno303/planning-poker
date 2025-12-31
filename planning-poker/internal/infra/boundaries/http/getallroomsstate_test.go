package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestGetAllRoomsStateAPI_Endpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockAdminHub(ctrl)
	api := NewGetAllRoomsStateAPI(mockHub, "test-api-key")

	if api.Endpoint() != "/admin/rooms" {
		t.Errorf("Endpoint() = %v, want %v", api.Endpoint(), "/admin/rooms")
	}
}

func TestGetAllRoomsStateAPI_Methods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockAdminHub(ctrl)
	api := NewGetAllRoomsStateAPI(mockHub, "test-api-key")
	methods := api.Methods()

	if len(methods) != 1 {
		t.Fatalf("Methods() length = %v, want %v", len(methods), 1)
	}
	if methods[0] != "GET" {
		t.Errorf("Methods()[0] = %v, want %v", methods[0], "GET")
	}
}

func TestGetAllRoomsStateAPI_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockAdminHub(ctrl)
	apiKey := "valid-api-key"
	api := NewGetAllRoomsStateAPI(mockHub, apiKey)

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

	rooms := []*entity.Room{
		{
			ID:      "room1",
			Clients: inmemory.NewInMemoryClientCollection(client1),
		},
		{
			ID:      "room2",
			Clients: inmemory.NewInMemoryClientCollection(client2),
		},
	}

	mockHub.EXPECT().
		GetRooms().
		Return(rooms)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	var response []GetAllRoomsStateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("response length = %v, want %v", len(response), 2)
	}

	if response[0].ID != "room1" {
		t.Errorf("response[0].ID = %v, want %v", response[0].ID, "room1")
	}
	if response[1].ID != "room2" {
		t.Errorf("response[1].ID = %v, want %v", response[1].ID, "room2")
	}
}

func TestGetAllRoomsStateAPI_Handle_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockAdminHub(ctrl)
	apiKey := "valid-api-key"
	api := NewGetAllRoomsStateAPI(mockHub, apiKey)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms", nil)
	req.Header.Set("Authorization", "Bearer wrong-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetAllRoomsStateAPI_Handle_EmptyRooms(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockAdminHub(ctrl)
	apiKey := "valid-api-key"
	api := NewGetAllRoomsStateAPI(mockHub, apiKey)

	mockHub.EXPECT().
		GetRooms().
		Return([]*entity.Room{})

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	var response []GetAllRoomsStateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("response length = %v, want %v", len(response), 0)
	}
}

func TestGetAllRoomsStateAPI_Handle_MissingAuthHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHub := domain.NewMockAdminHub(ctrl)
	apiKey := "valid-api-key"
	api := NewGetAllRoomsStateAPI(mockHub, apiKey)

	handler := api.Handle()
	req := httptest.NewRequest(http.MethodGet, "/admin/rooms", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}
