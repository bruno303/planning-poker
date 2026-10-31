package bus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/domainerror"
	"planning-poker/internal/domain/entity"

	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

const (
	commandTestClientID  = "client-1"
	commandTestRoomID    = "room-1"
	commandTestCommandID = "6f1a0f9e-1a2b-4c3d-8e4f-5a6b7c8d9e0f"
)

type decodedCommandFrame struct {
	Type        string  `json:"type"`
	CommandID   string  `json:"commandId"`
	Code        string  `json:"code"`
	Message     string  `json:"message"`
	RoomVersion *uint64 `json:"roomVersion"`
}

func newCommandTestBus(t *testing.T, hub domain.Hub, calls map[string]useCaseCall, ledger CommandLedger) (*WebsocketBus, *websocket.Conn) {
	t.Helper()

	serverConn, clientConn := websocketPair(t)
	bus := NewWebsocketBus(
		commandTestClientID,
		commandTestRoomID,
		serverConn,
		hub,
		usecase.UseCasesFacade{},
		ledger,
		WebSocketConfig{WriteTimeout: time.Second, ReadTimeout: time.Second, PingInterval: time.Hour},
	)
	bus.calls = calls
	bus.Detach()
	t.Cleanup(func() {
		_ = bus.Close()
		_ = clientConn.Close()
	})

	return bus, clientConn
}

type fakeCommandLedger struct {
	mu          sync.Mutex
	outcomes    map[string][]byte
	pending     map[string]bool
	claimErr    error
	completeErr error
	claims      int
	completes   int
	lastOutcome []byte
}

func newFakeCommandLedger() *fakeCommandLedger {
	return &fakeCommandLedger{
		outcomes: map[string][]byte{},
		pending:  map[string]bool{},
	}
}

func (f *fakeCommandLedger) key(roomID, clientID, commandID string) string {
	return roomID + ":" + clientID + ":" + commandID
}

func (f *fakeCommandLedger) Claim(_ context.Context, roomID, clientID, commandID string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.claims++
	if f.claimErr != nil {
		return nil, false, f.claimErr
	}

	key := f.key(roomID, clientID, commandID)
	if outcome, ok := f.outcomes[key]; ok {
		return outcome, true, nil
	}
	if f.pending[key] {
		return nil, false, ErrCommandInProgress
	}

	f.pending[key] = true
	return nil, false, nil
}

func (f *fakeCommandLedger) Complete(_ context.Context, roomID, clientID, commandID string, outcome []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.completes++
	if f.completeErr != nil {
		return f.completeErr
	}

	key := f.key(roomID, clientID, commandID)
	delete(f.pending, key)
	f.outcomes[key] = outcome
	f.lastOutcome = outcome
	return nil
}

func (f *fakeCommandLedger) claimCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claims
}

func (f *fakeCommandLedger) completeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.completes
}

func readCommandFrame(t *testing.T, conn *websocket.Conn) decodedCommandFrame {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}

	var frame decodedCommandFrame
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatalf("failed to read command frame: %v", err)
	}

	return frame
}

func assertNoCommandFrame(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}

	var raw json.RawMessage
	err := conn.ReadJSON(&raw)
	if err == nil {
		t.Fatalf("expected no frame, got %s", raw)
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected read timeout while waiting for a frame, got %v", err)
	}
}

func TestWebsocketBus_Process_SendsCommandAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	roomVersion := uint64(12)

	hub := domain.NewMockHub(ctrl)
	hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(&entity.Room{RoomVersion: roomVersion}, nil)

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), usecase.UpdateNameCommand{
		RoomID:   commandTestRoomID,
		SenderID: commandTestClientID,
		Username: "Alice",
	}).Return(nil)

	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), newFakeCommandLedger())

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandAckType {
		t.Fatalf("expected frame type %q, got %q", commandAckType, frame.Type)
	}
	if frame.CommandID != commandTestCommandID {
		t.Fatalf("expected commandId %q, got %q", commandTestCommandID, frame.CommandID)
	}
	if frame.RoomVersion == nil || *frame.RoomVersion != roomVersion {
		t.Fatalf("expected roomVersion %d, got %v", roomVersion, frame.RoomVersion)
	}
	if frame.Code != "" || frame.Message != "" {
		t.Fatalf("expected an ack without error fields, got %+v", frame)
	}
}

func TestWebsocketBus_Process_SendsCommandAckWhenRoomLoadFails(t *testing.T) {
	ctrl := gomock.NewController(t)

	hub := domain.NewMockHub(ctrl)
	hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(nil, errors.New("hub unavailable"))

	vote := usecase.NewMockUseCase[usecase.VoteCommand](ctrl)
	vote.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil)

	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{Vote: vote},
		commandTestClientID,
		commandTestRoomID,
	), newFakeCommandLedger())

	bus.process(context.Background(), WebSocketMessage{
		Type:      "vote",
		CommandID: commandTestCommandID,
		Payload:   VotePayload{Vote: "5"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandAckType {
		t.Fatalf("expected frame type %q, got %q", commandAckType, frame.Type)
	}
	if frame.CommandID != commandTestCommandID {
		t.Fatalf("expected commandId %q, got %q", commandTestCommandID, frame.CommandID)
	}
	if frame.RoomVersion != nil {
		t.Fatalf("expected no roomVersion when the load fails, got %d", *frame.RoomVersion)
	}
}

func TestWebsocketBus_Process_SendsCommandErrorForStaleVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	roomVersion := uint64(9)

	hub := domain.NewMockHub(ctrl)
	hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(&entity.Room{RoomVersion: roomVersion}, nil)

	reset := usecase.NewMockUseCase[usecase.ResetCommand](ctrl)
	reset.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(domain.ErrStaleRoomVersion)

	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{Reset: reset},
		commandTestClientID,
		commandTestRoomID,
	), newFakeCommandLedger())

	bus.process(context.Background(), WebSocketMessage{
		Type:      "reset",
		CommandID: commandTestCommandID,
		Payload:   map[string]uint64{"expectedRoomVersion": 5},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.CommandID != commandTestCommandID {
		t.Fatalf("expected commandId %q, got %q", commandTestCommandID, frame.CommandID)
	}
	if frame.Code != wireCodeStaleVersion {
		t.Fatalf("expected code %q, got %q", wireCodeStaleVersion, frame.Code)
	}
	if frame.Message != "stale room version" {
		t.Fatalf("expected stale message, got %q", frame.Message)
	}
	if frame.RoomVersion == nil || *frame.RoomVersion != roomVersion {
		t.Fatalf("expected roomVersion %d, got %v", roomVersion, frame.RoomVersion)
	}
}

func TestWebsocketBus_Process_MapsDomainErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     domainerror.Code
		wantCode string
		message  string
	}{
		{name: "forbidden", code: domainerror.CodeForbidden, wantCode: wireCodeForbidden, message: "only the room owner can reset voting"},
		{name: "not_found", code: domainerror.CodeNotFound, wantCode: wireCodeNotFound, message: "room not found"},
		{name: "validation", code: domainerror.CodeValidation, wantCode: wireCodeValidation, message: "story ID cannot be empty"},
		{name: "conflict", code: domainerror.CodeConflict, wantCode: wireCodeConflict, message: "story story-1 is already the current story"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			hub := domain.NewMockHub(ctrl)

			updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
			updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(domainerror.Newf(tt.code, "%s", tt.message))

			bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
				usecase.UseCasesFacade{UpdateName: updateName},
				commandTestClientID,
				commandTestRoomID,
			), newFakeCommandLedger())

			bus.process(context.Background(), WebSocketMessage{
				Type:      "update-name",
				CommandID: commandTestCommandID,
				Payload:   UpdateNamePayload{Username: "Alice"},
			})

			frame := readCommandFrame(t, clientConn)
			if frame.Type != commandErrorType {
				t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
			}
			if frame.Code != tt.wantCode {
				t.Fatalf("expected code %q, got %q", tt.wantCode, frame.Code)
			}
			if frame.Message != tt.message {
				t.Fatalf("expected message %q, got %q", tt.message, frame.Message)
			}
			if frame.RoomVersion != nil {
				t.Fatalf("expected no roomVersion for code %q, got %d", tt.wantCode, *frame.RoomVersion)
			}
		})
	}
}

func TestWebsocketBus_Process_HidesInternalErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	hub := domain.NewMockHub(ctrl)

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(errors.New("database exploded"))

	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), newFakeCommandLedger())

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.Code != wireCodeInternal {
		t.Fatalf("expected code %q, got %q", wireCodeInternal, frame.Code)
	}
	if frame.Message != "internal error" {
		t.Fatalf("expected message %q, got %q", "internal error", frame.Message)
	}
}

func TestWebsocketBus_Process_UnknownCommandWithCommandID(t *testing.T) {
	ctrl := gomock.NewController(t)
	bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), map[string]useCaseCall{}, newFakeCommandLedger())

	bus.process(context.Background(), WebSocketMessage{
		Type:      "does-not-exist",
		CommandID: commandTestCommandID,
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.CommandID != commandTestCommandID {
		t.Fatalf("expected commandId %q, got %q", commandTestCommandID, frame.CommandID)
	}
	if frame.Code != wireCodeUnknownCommand {
		t.Fatalf("expected code %q, got %q", wireCodeUnknownCommand, frame.Code)
	}
	if frame.Message != "unknown command type" {
		t.Fatalf("expected message %q, got %q", "unknown command type", frame.Message)
	}
}

func TestWebsocketBus_Process_InvalidCommandID(t *testing.T) {
	ctrl := gomock.NewController(t)
	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), newFakeCommandLedger())

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: "not-a-uuid",
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.CommandID != "not-a-uuid" {
		t.Fatalf("expected commandId %q, got %q", "not-a-uuid", frame.CommandID)
	}
	if frame.Code != wireCodeInvalidCommandID {
		t.Fatalf("expected code %q, got %q", wireCodeInvalidCommandID, frame.Code)
	}
	if frame.Message != "commandId must be a UUID" {
		t.Fatalf("expected message %q, got %q", "commandId must be a UUID", frame.Message)
	}
}

func TestWebsocketBus_Process_InvalidPayload(t *testing.T) {
	t.Run("unserializable payload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
		updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

		bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
			usecase.UseCasesFacade{UpdateName: updateName},
			commandTestClientID,
			commandTestRoomID,
		), newFakeCommandLedger())

		bus.process(context.Background(), WebSocketMessage{
			Type:      "update-name",
			CommandID: commandTestCommandID,
			Payload:   func() {},
		})

		assertInvalidPayloadFrame(t, clientConn)
	})

	t.Run("malformed expected room version", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reset := usecase.NewMockUseCase[usecase.ResetCommand](ctrl)
		reset.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

		bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
			usecase.UseCasesFacade{Reset: reset},
			commandTestClientID,
			commandTestRoomID,
		), newFakeCommandLedger())

		bus.process(context.Background(), WebSocketMessage{
			Type:      "reset",
			CommandID: commandTestCommandID,
			Payload:   map[string]string{"expectedRoomVersion": "abc"},
		})

		assertInvalidPayloadFrame(t, clientConn)
	})
}

func assertInvalidPayloadFrame(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	frame := readCommandFrame(t, conn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.CommandID != commandTestCommandID {
		t.Fatalf("expected commandId %q, got %q", commandTestCommandID, frame.CommandID)
	}
	if frame.Code != wireCodeInvalidPayload {
		t.Fatalf("expected code %q, got %q", wireCodeInvalidPayload, frame.Code)
	}
	if frame.Message != "invalid payload" {
		t.Fatalf("expected message %q, got %q", "invalid payload", frame.Message)
	}
}

func readRawCommandFrame(t *testing.T, conn *websocket.Conn) []byte {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}

	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read raw command frame: %v", err)
	}

	return payload
}

func decodeCommandFrame(t *testing.T, payload []byte) decodedCommandFrame {
	t.Helper()

	var frame decodedCommandFrame
	if err := json.Unmarshal(payload, &frame); err != nil {
		t.Fatalf("failed to decode command frame %s: %v", payload, err)
	}

	return frame
}

func TestWebsocketBus_Process_CachesCommandAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	roomVersion := uint64(12)

	hub := domain.NewMockHub(ctrl)
	hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(&entity.Room{RoomVersion: roomVersion}, nil)

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil)

	ledger := newFakeCommandLedger()
	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), ledger)

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	payload := readRawCommandFrame(t, clientConn)
	frame := decodeCommandFrame(t, payload)
	if frame.Type != commandAckType {
		t.Fatalf("expected frame type %q, got %q", commandAckType, frame.Type)
	}

	if ledger.completeCount() != 1 {
		t.Fatalf("expected the outcome to be stored once, got %d", ledger.completeCount())
	}
	if !bytes.Equal(ledger.lastOutcome, payload) {
		t.Fatalf("expected stored outcome %s to match the sent frame %s", ledger.lastOutcome, payload)
	}
}

func TestWebsocketBus_Process_ReplaysCompletedCommandOutcome(t *testing.T) {
	ctrl := gomock.NewController(t)
	roomVersion := uint64(12)

	hub := domain.NewMockHub(ctrl)
	hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(&entity.Room{RoomVersion: roomVersion}, nil)

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	ledger := newFakeCommandLedger()
	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), ledger)

	message := WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	}

	bus.process(context.Background(), message)
	first := readRawCommandFrame(t, clientConn)

	bus.process(context.Background(), message)
	second := readRawCommandFrame(t, clientConn)

	if !bytes.Equal(first, second) {
		t.Fatalf("expected the duplicate command to replay %s, got %s", first, second)
	}
	if ledger.claimCount() != 2 {
		t.Fatalf("expected two claims, got %d", ledger.claimCount())
	}
	if ledger.completeCount() != 1 {
		t.Fatalf("expected the outcome to be stored once, got %d", ledger.completeCount())
	}
}

func TestWebsocketBus_Process_CommandInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	ledger := newFakeCommandLedger()
	ledger.pending[ledger.key(commandTestRoomID, commandTestClientID, commandTestCommandID)] = true

	bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), ledger)

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.Code != wireCodeCommandInProgress {
		t.Fatalf("expected code %q, got %q", wireCodeCommandInProgress, frame.Code)
	}
	if frame.Message != "command is already being processed" {
		t.Fatalf("expected message %q, got %q", "command is already being processed", frame.Message)
	}
}

func TestWebsocketBus_Process_ClaimFailureReturnsInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	ledger := newFakeCommandLedger()
	ledger.claimErr = errors.New("redis unavailable")

	bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), ledger)

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.Code != wireCodeInternal {
		t.Fatalf("expected code %q, got %q", wireCodeInternal, frame.Code)
	}
	if frame.Message != "internal error" {
		t.Fatalf("expected message %q, got %q", "internal error", frame.Message)
	}
}

func TestWebsocketBus_Process_CompleteFailureStillSendsFrame(t *testing.T) {
	ctrl := gomock.NewController(t)
	roomVersion := uint64(3)

	hub := domain.NewMockHub(ctrl)
	hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(&entity.Room{RoomVersion: roomVersion}, nil)

	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil)

	ledger := newFakeCommandLedger()
	ledger.completeErr = errors.New("redis unavailable")

	bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), ledger)

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandAckType {
		t.Fatalf("expected frame type %q, got %q", commandAckType, frame.Type)
	}
	if ledger.completeCount() != 1 {
		t.Fatalf("expected one completion attempt, got %d", ledger.completeCount())
	}
}

func TestWebsocketBus_Process_NilLedgerReturnsInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
	updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
		usecase.UseCasesFacade{UpdateName: updateName},
		commandTestClientID,
		commandTestRoomID,
	), nil)

	bus.process(context.Background(), WebSocketMessage{
		Type:      "update-name",
		CommandID: commandTestCommandID,
		Payload:   UpdateNamePayload{Username: "Alice"},
	})

	frame := readCommandFrame(t, clientConn)
	if frame.Type != commandErrorType {
		t.Fatalf("expected frame type %q, got %q", commandErrorType, frame.Type)
	}
	if frame.Code != wireCodeInternal {
		t.Fatalf("expected code %q, got %q", wireCodeInternal, frame.Code)
	}
	if frame.Message != "internal error" {
		t.Fatalf("expected message %q, got %q", "internal error", frame.Message)
	}
}

func TestWebsocketBus_Process_LegacyCommandSendsNoAck(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		updateName := usecase.NewMockUseCase[usecase.UpdateNameCommand](ctrl)
		updateName.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil)

		ledger := newFakeCommandLedger()
		bus, clientConn := newCommandTestBus(t, domain.NewMockHub(ctrl), mapUsecases(
			usecase.UseCasesFacade{UpdateName: updateName},
			commandTestClientID,
			commandTestRoomID,
		), ledger)

		bus.process(context.Background(), WebSocketMessage{
			Type:    "update-name",
			Payload: UpdateNamePayload{Username: "Alice"},
		})

		assertNoCommandFrame(t, clientConn)
		if ledger.claimCount() != 0 {
			t.Fatalf("expected legacy commands to bypass the ledger, got %d claims", ledger.claimCount())
		}
	})

	t.Run("stale version", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		roomVersion := uint64(6)

		hub := domain.NewMockHub(ctrl)
		hub.EXPECT().LoadRoom(gomock.Any(), commandTestRoomID).Return(&entity.Room{RoomVersion: roomVersion}, nil)

		reset := usecase.NewMockUseCase[usecase.ResetCommand](ctrl)
		reset.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(domain.ErrStaleRoomVersion)

		bus, clientConn := newCommandTestBus(t, hub, mapUsecases(
			usecase.UseCasesFacade{Reset: reset},
			commandTestClientID,
			commandTestRoomID,
		), newFakeCommandLedger())

		bus.process(context.Background(), WebSocketMessage{
			Type:    "reset",
			Payload: map[string]uint64{"expectedRoomVersion": 5},
		})

		assertStaleCommand(t, clientConn, true, roomVersion)
	})
}
