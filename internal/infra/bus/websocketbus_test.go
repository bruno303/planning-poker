package bus

import (
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

func TestWebsocketBus_Detach_PreventsLeaveRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverCh <- conn
		<-make(chan struct{})
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial test websocket: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverCh

	mockLeaveRoom := usecase.NewMockUseCase[usecase.LeaveRoomCommand](ctrl)
	mockLeaveRoom.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	usecases := usecase.UseCasesFacade{
		LeaveRoom: mockLeaveRoom,
	}

	bus := NewWebsocketBus(
		"test-client",
		"test-room",
		serverConn,
		domain.NewMockHub(ctrl),
		usecases,
		WebSocketConfig{},
	)

	bus.Detach()
	if err := bus.Close(); err != nil {
		t.Fatalf("expected no error from Close after Detach, got %v", err)
	}
}

func TestWebsocketBus_Close_WithoutDetach_CallsLeaveRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverCh <- conn
		<-make(chan struct{})
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial test websocket: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverCh

	mockLeaveRoom := usecase.NewMockUseCase[usecase.LeaveRoomCommand](ctrl)
	mockLeaveRoom.EXPECT().
		Execute(gomock.Any(), usecase.LeaveRoomCommand{
			RoomID:   "test-room",
			SenderID: "test-client",
		}).
		Return(nil)

	usecases := usecase.UseCasesFacade{
		LeaveRoom: mockLeaveRoom,
	}

	bus := NewWebsocketBus(
		"test-client",
		"test-room",
		serverConn,
		domain.NewMockHub(ctrl),
		usecases,
		WebSocketConfig{},
	)

	if err := bus.Close(); err != nil {
		t.Fatalf("expected no error from Close, got %v", err)
	}
}

func TestWebsocketBus_Detach_IsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverCh <- conn
		<-make(chan struct{})
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial test websocket: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverCh

	mockLeaveRoom := usecase.NewMockUseCase[usecase.LeaveRoomCommand](ctrl)
	mockLeaveRoom.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	usecases := usecase.UseCasesFacade{
		LeaveRoom: mockLeaveRoom,
	}

	bus := NewWebsocketBus(
		"test-client",
		"test-room",
		serverConn,
		domain.NewMockHub(ctrl),
		usecases,
		WebSocketConfig{},
	)

	bus.Detach()
	bus.Detach()
	if err := bus.Close(); err != nil {
		t.Fatalf("expected no error from Close after double Detach, got %v", err)
	}
}
