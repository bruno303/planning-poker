package bus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

func TestMapUsecases_DispatchesSupportedCommands(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	clientID := "client-123"
	roomID := "room-123"

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	vote := usecase.NewMockUseCase[usecase.VoteCommand](ctrl)
	reset := usecase.NewMockUseCase[usecase.ResetCommand](ctrl)
	reveal := usecase.NewMockUseCase[usecase.RevealCommand](ctrl)
	toggleSpectator := usecase.NewMockUseCase[usecase.ToggleSpectatorCommand](ctrl)
	toggleOwner := usecase.NewMockUseCase[usecase.ToggleOwnerCommand](ctrl)
	updateStory := usecase.NewMockUseCase[usecase.UpdateStoryCommand](ctrl)
	newVoting := usecase.NewMockUseCase[usecase.NewVotingCommand](ctrl)
	voteAgain := usecase.NewMockUseCase[usecase.VoteAgainCommand](ctrl)
	toggleBacklogMode := usecase.NewMockUseCase[usecase.ToggleBacklogModeCommand](ctrl)
	addStory := usecase.NewMockUseCase[usecase.AddStoryCommand](ctrl)
	removeStory := usecase.NewMockUseCase[usecase.RemoveStoryCommand](ctrl)
	advanceStory := usecase.NewMockUseCase[usecase.AdvanceStoryCommand](ctrl)
	prevStory := usecase.NewMockUseCase[usecase.PrevStoryCommand](ctrl)
	selectStory := usecase.NewMockUseCase[usecase.SelectStoryCommand](ctrl)
	reorderStory := usecase.NewMockUseCase[usecase.ReorderStoryCommand](ctrl)

	voteValue := "5"
	commands := []struct {
		name string
		msg  WebSocketMessage
		set  func()
	}{
		{
			name: "update-name",
			msg:  WebSocketMessage{Type: "update-name", Payload: UpdateNamePayload{Username: "Alice"}},
			set: func() {
				updateName.EXPECT().Execute(gomock.Any(), usecase.UpdateNameCommand{RoomID: roomID, SenderID: clientID, Username: "Alice"}).Return(nil)
			},
		},
		{
			name: "vote",
			msg:  WebSocketMessage{Type: "vote", Payload: VotePayload{Vote: voteValue}},
			set: func() {
				vote.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, cmd usecase.VoteCommand) error {
					if cmd.RoomID != roomID || cmd.SenderID != clientID || cmd.Vote == nil || *cmd.Vote != voteValue {
						t.Errorf("unexpected vote command: %#v", cmd)
					}
					return nil
				})
			},
		},
		{
			name: "reset",
			msg:  WebSocketMessage{Type: "reset"},
			set: func() {
				reset.EXPECT().Execute(gomock.Any(), usecase.ResetCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "reveal-votes",
			msg:  WebSocketMessage{Type: "reveal-votes"},
			set: func() {
				reveal.EXPECT().Execute(gomock.Any(), usecase.RevealCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "toggle-spectator",
			msg:  WebSocketMessage{Type: "toggle-spectator", Payload: ToggleSpectatorPayload{TargetClientID: "target-1"}},
			set: func() {
				toggleSpectator.EXPECT().Execute(gomock.Any(), usecase.ToggleSpectatorCommand{RoomID: roomID, SenderID: clientID, TargetClientID: "target-1"}).Return(nil)
			},
		},
		{
			name: "toggle-owner",
			msg:  WebSocketMessage{Type: "toggle-owner", Payload: ToggleOwnerPayload{TargetClientID: "target-2"}},
			set: func() {
				toggleOwner.EXPECT().Execute(gomock.Any(), usecase.ToggleOwnerCommand{RoomID: roomID, SenderID: clientID, TargetClientID: "target-2"}).Return(nil)
			},
		},
		{
			name: "update-story",
			msg:  WebSocketMessage{Type: "update-story", Payload: UpdateStoryPayload{Story: "Current story"}},
			set: func() {
				updateStory.EXPECT().Execute(gomock.Any(), usecase.UpdateStoryCommand{RoomID: roomID, SenderID: clientID, Story: "Current story"}).Return(nil)
			},
		},
		{
			name: "new-voting",
			msg:  WebSocketMessage{Type: "new-voting"},
			set: func() {
				newVoting.EXPECT().Execute(gomock.Any(), usecase.NewVotingCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "vote-again",
			msg:  WebSocketMessage{Type: "vote-again"},
			set: func() {
				voteAgain.EXPECT().Execute(gomock.Any(), usecase.VoteAgainCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "toggle-backlog-mode",
			msg:  WebSocketMessage{Type: "toggle-backlog-mode"},
			set: func() {
				toggleBacklogMode.EXPECT().Execute(gomock.Any(), usecase.ToggleBacklogModeCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "add-story",
			msg:  WebSocketMessage{Type: "add-story", Payload: AddStoryPayload{Story: "Story from backlog"}},
			set: func() {
				addStory.EXPECT().Execute(gomock.Any(), usecase.AddStoryCommand{RoomID: roomID, SenderID: clientID, StoryName: "Story from backlog"}).Return(nil)
			},
		},
		{
			name: "remove-story",
			msg:  WebSocketMessage{Type: "remove-story", Payload: RemoveStoryPayload{StoryID: "story-3"}},
			set: func() {
				removeStory.EXPECT().Execute(gomock.Any(), usecase.RemoveStoryCommand{
					RoomID: roomID, SenderID: clientID, StoryID: "story-3",
				}).Return(nil)
			},
		},
		{
			name: "advance-story",
			msg:  WebSocketMessage{Type: "advance-story"},
			set: func() {
				advanceStory.EXPECT().Execute(gomock.Any(), usecase.AdvanceStoryCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "prev-story",
			msg:  WebSocketMessage{Type: "prev-story"},
			set: func() {
				prevStory.EXPECT().Execute(gomock.Any(), usecase.PrevStoryCommand{RoomID: roomID, SenderID: clientID}).Return(nil)
			},
		},
		{
			name: "select-story",
			msg:  WebSocketMessage{Type: "select-story", Payload: SelectStoryPayload{StoryID: "story-1"}},
			set: func() {
				selectStory.EXPECT().Execute(gomock.Any(), usecase.SelectStoryCommand{RoomID: roomID, SenderID: clientID, StoryID: "story-1"}).Return(nil)
			},
		},
		{
			name: "reorder-story",
			msg: WebSocketMessage{Type: "reorder-story", Payload: ReorderStoryPayload{
				StoryID: "story-2", TargetIndex: 1,
			}},
			set: func() {
				reorderStory.EXPECT().Execute(gomock.Any(), usecase.ReorderStoryCommand{
					RoomID: roomID, SenderID: clientID, StoryID: "story-2", TargetIndex: 1,
				}).Return(nil)
			},
		},
	}

	usecases := usecase.UseCasesFacade{
		UpdateName:        updateName,
		Vote:              vote,
		Reset:             reset,
		Reveal:            reveal,
		ToggleSpectator:   toggleSpectator,
		ToggleOwner:       toggleOwner,
		UpdateStory:       updateStory,
		NewVoting:         newVoting,
		VoteAgain:         voteAgain,
		ToggleBacklogMode: toggleBacklogMode,
		AddStory:          addStory,
		RemoveStory:       removeStory,
		AdvanceStory:      advanceStory,
		PrevStory:         prevStory,
		SelectStory:       selectStory,
		ReorderStory:      reorderStory,
	}
	calls := mapUsecases(usecases, clientID, roomID)

	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			command.set()
			if err := calls[command.msg.Type](ctx, command.msg); err != nil {
				t.Fatalf("dispatch returned error: %v", err)
			}
		})
	}
}

func TestMapUsecases_InvalidPayloadReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	calls := mapUsecases(usecase.UseCasesFacade{UpdateName: updateName}, "client-1", "room-1")

	err := calls["update-name"](context.Background(), WebSocketMessage{
		Type:    "update-name",
		Payload: func() {},
	})

	if err == nil || err.Error() != "invalid payload" {
		t.Fatalf("expected invalid payload error, got %v", err)
	}
}

func TestWebsocketBus_Process_UnknownEventDoesNotDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	bus := &WebsocketBus{
		ID:     "client-1",
		logger: log.NewLogger("test"),
		calls:  mapUsecases(usecase.UseCasesFacade{UpdateName: updateName}, "client-1", "room-1"),
	}

	bus.process(context.Background(), WebSocketMessage{Type: "unknown-event"})
}

func TestWebsocketBus_Process_WhenUseCaseFailsDoesNotStopBus(t *testing.T) {
	called := false
	bus := &WebsocketBus{
		ID:     "client-1",
		logger: log.NewLogger("test"),
		calls: map[string]useCaseCall{
			"event": func(context.Context, WebSocketMessage) error {
				called = true
				return errors.New("use case failed")
			},
		},
	}

	bus.process(context.Background(), WebSocketMessage{Type: "event"})
	if !called {
		t.Fatal("expected use case callback to be dispatched")
	}
}

func TestWebsocketBus_Listen_DispatchesIncomingMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	serverConn, clientConn := websocketPair(t)

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), usecase.UpdateNameCommand{
		RoomID: "room-1", SenderID: "client-1", Username: "Alice",
	}).Return(nil)
	leaveRoom := usecase.NewMockUseCase[usecase.LeaveRoomCommand](ctrl)
	leaveRoom.EXPECT().Execute(gomock.Any(), usecase.LeaveRoomCommand{RoomID: "room-1", SenderID: "client-1"}).Return(nil)

	bus := NewWebsocketBus(
		"client-1",
		"room-1",
		serverConn,
		domain.NewMockHub(ctrl),
		usecase.UseCasesFacade{UpdateName: updateName, LeaveRoom: leaveRoom},
		WebSocketConfig{ReadTimeout: time.Second, WriteTimeout: time.Second, PingInterval: time.Second},
	)

	done := make(chan struct{})
	go func() {
		bus.Listen(context.Background())
		close(done)
	}()

	if err := clientConn.WriteJSON(WebSocketMessage{Type: "update-name", Payload: UpdateNamePayload{Username: "Alice"}}); err != nil {
		t.Fatalf("failed to send websocket message: %v", err)
	}
	if err := clientConn.Close(); err != nil {
		t.Fatalf("failed to close client websocket: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("bus did not stop after client connection closed")
	}
}

func TestWebsocketBus_Send_WritesMessage(t *testing.T) {
	serverConn, clientConn := websocketPair(t)
	defer clientConn.Close()

	bus := NewWebsocketBus("client-1", "room-1", serverConn, domain.NewMockHub(gomock.NewController(t)), usecase.UseCasesFacade{}, WebSocketConfig{WriteTimeout: time.Second})
	bus.Detach()
	defer bus.Close()

	want := WebSocketMessage{Type: "room-state", Payload: map[string]string{"state": "ready"}}
	if err := bus.Send(context.Background(), want); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	var got WebSocketMessage
	if err := clientConn.ReadJSON(&got); err != nil {
		t.Fatalf("failed to read sent message: %v", err)
	}
	if got.Type != want.Type {
		t.Fatalf("expected message type %q, got %q", want.Type, got.Type)
	}
}

func TestWebsocketBus_Send_WhenConnectionClosedReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	serverConn, clientConn := websocketPair(t)
	defer clientConn.Close()

	bus := NewWebsocketBus("client-1", "room-1", serverConn, domain.NewMockHub(ctrl), usecase.UseCasesFacade{}, WebSocketConfig{})
	bus.Detach()
	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	if err := bus.Send(context.Background(), "message"); err == nil {
		t.Fatal("expected sending on a closed bus to fail")
	}
}

func TestWebsocketBus_Send_WhenSocketWriteFailsReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	serverConn, clientConn := websocketPair(t)
	defer clientConn.Close()

	bus := NewWebsocketBus("client-1", "room-1", serverConn, domain.NewMockHub(ctrl), usecase.UseCasesFacade{}, WebSocketConfig{})
	bus.Detach()
	if err := serverConn.Close(); err != nil {
		t.Fatalf("failed to close server connection: %v", err)
	}

	if err := bus.Send(context.Background(), "message"); err == nil {
		t.Fatal("expected socket write failure")
	}
}

func websocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
	}))
	t.Cleanup(server.Close)

	url := "ws://" + strings.TrimPrefix(server.URL, "http://")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}

	select {
	case serverConn := <-serverConnCh:
		return serverConn, clientConn
	case <-time.After(time.Second):
		clientConn.Close()
		t.Fatal("server did not accept websocket connection")
		return nil, nil
	}
}
