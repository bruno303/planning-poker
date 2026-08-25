package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/bus"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

func TestWebsocketAPI_Handle_MissingRoomIDReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewWebsocketAPI(usecase.UseCasesFacade{}, bus.NewWebSocketBusFactory(domain.NewMockHub(ctrl), usecase.UseCasesFacade{}, bus.WebSocketConfig{}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/planning//ws", nil)

	api.Handle().ServeHTTP(recorder, request)

	assertJSONErrorResponse(t, recorder, http.StatusBadRequest, `{"error":"Room ID is required"}`)
}

func TestWebsocketAPI_Handle_WhenUpgradeFailsReturnsBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewWebsocketAPI(usecase.UseCasesFacade{}, bus.NewWebSocketBusFactory(domain.NewMockHub(ctrl), usecase.UseCasesFacade{}, bus.WebSocketConfig{}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/planning/room-1/ws", nil)
	request = mux.SetURLVars(request, map[string]string{"roomID": "room-1"})

	api.Handle().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestWebsocketAPI_Handle_WhenCreateClientFailsSendsWebsocketError(t *testing.T) {
	ctrl := gomock.NewController(t)
	createClient := usecase.NewMockUseCaseO[usecase.CreateClientOutput](ctrl)
	createClient.EXPECT().Execute(gomock.Any()).Return(usecase.CreateClientOutput{}, errors.New("cannot create client"))

	api := NewWebsocketAPI(
		usecase.UseCasesFacade{CreateClient: createClient},
		bus.NewWebSocketBusFactory(domain.NewMockHub(ctrl), usecase.UseCasesFacade{}, bus.WebSocketConfig{}),
	)
	server := newWebsocketAPITestServer(t, api)
	defer server.Close()

	conn := dialWebsocketAPI(t, server, "/planning/room-1/ws")
	defer conn.Close()

	assertWebsocketInternalError(t, conn)
}

func TestWebsocketAPI_Handle_WhenJoinFailsSendsWebsocketError(t *testing.T) {
	ctrl := gomock.NewController(t)
	joinRoom := usecase.NewMockUseCaseR[usecase.JoinRoomCommand, *usecase.JoinRoomOutput](ctrl)
	joinRoom.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, cmd usecase.JoinRoomCommand) (*usecase.JoinRoomOutput, error) {
		if cmd.RoomID != "room-1" || cmd.SenderID != "client-1" || cmd.Bus == nil {
			t.Errorf("unexpected join command: %#v", cmd)
		}
		return nil, errors.New("room unavailable")
	})

	api := NewWebsocketAPI(
		usecase.UseCasesFacade{JoinRoom: joinRoom},
		bus.NewWebSocketBusFactory(domain.NewMockHub(ctrl), usecase.UseCasesFacade{}, bus.WebSocketConfig{}),
	)
	server := newWebsocketAPITestServer(t, api)
	defer server.Close()

	conn := dialWebsocketAPI(t, server, "/planning/room-1/ws?clientId=client-1")
	defer conn.Close()

	assertWebsocketInternalError(t, conn)
}

func newWebsocketAPITestServer(t *testing.T, api *WebsocketAPI) *httptest.Server {
	t.Helper()
	router := mux.NewRouter()
	router.Handle(api.Endpoint(), api.Handle())
	return httptest.NewServer(router)
}

func dialWebsocketAPI(t *testing.T, server *httptest.Server, path string) *websocket.Conn {
	t.Helper()
	url := "ws" + server.URL[len("http"):]
	url += path
	conn, response, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		if response != nil {
			t.Fatalf("failed to dial websocket, status %d: %v", response.StatusCode, err)
		}
		t.Fatalf("failed to dial websocket: %v", err)
	}
	return conn
}

func assertWebsocketInternalError(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatal("expected websocket close error")
	}
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) {
		t.Fatalf("expected websocket close error, got %T: %v", err, err)
	}
	if closeErr.Code != websocket.CloseInternalServerErr {
		t.Fatalf("expected close code %d, got %d", websocket.CloseInternalServerErr, closeErr.Code)
	}
}
