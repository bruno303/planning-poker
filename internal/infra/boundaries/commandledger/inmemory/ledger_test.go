package inmemory

import (
	"context"
	"errors"
	"testing"
	"time"

	"planning-poker/internal/infra/bus"
)

const (
	testRoomID    = "room-1"
	testClientID  = "client-1"
	testCommandID = "cmd-1"
)

func TestLedger_ClaimReservesCommand(t *testing.T) {
	ledger := New(bus.CommandPendingTTL, bus.CommandResultTTL)

	outcome, found, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID)
	if err != nil {
		t.Fatalf("expected no error from the first claim, got %v", err)
	}
	if found {
		t.Fatalf("expected the first claim to acquire the reservation, got outcome %s", outcome)
	}

	_, found, err = ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID)
	if !errors.Is(err, bus.ErrCommandInProgress) {
		t.Fatalf("expected ErrCommandInProgress for a concurrent claim, got %v", err)
	}
	if found {
		t.Fatal("expected no outcome while the command is pending")
	}
}

func TestLedger_CompleteReplaysOutcome(t *testing.T) {
	ledger := New(bus.CommandPendingTTL, bus.CommandResultTTL)
	payload := []byte(`{"type":"command-ack"}`)

	if _, _, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID); err != nil {
		t.Fatalf("expected claim to succeed, got %v", err)
	}
	if err := ledger.Complete(context.Background(), testRoomID, testClientID, testCommandID, payload); err != nil {
		t.Fatalf("expected complete to succeed, got %v", err)
	}

	outcome, found, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID)
	if err != nil {
		t.Fatalf("expected no error replaying the outcome, got %v", err)
	}
	if !found {
		t.Fatal("expected the completed outcome to be found")
	}
	if string(outcome) != string(payload) {
		t.Fatalf("expected outcome %s, got %s", payload, outcome)
	}
}

func TestLedger_PendingEntryExpires(t *testing.T) {
	now := time.Now()
	ledger := New(bus.CommandPendingTTL, bus.CommandResultTTL)
	ledger.now = func() time.Time { return now }

	if _, _, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID); err != nil {
		t.Fatalf("expected claim to succeed, got %v", err)
	}

	now = now.Add(bus.CommandPendingTTL + time.Second)

	outcome, found, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID)
	if err != nil {
		t.Fatalf("expected the expired reservation to be reclaimable, got %v", err)
	}
	if found {
		t.Fatalf("expected no outcome after the pending entry expired, got %s", outcome)
	}
}

func TestLedger_CompletedEntryExpires(t *testing.T) {
	now := time.Now()
	ledger := New(bus.CommandPendingTTL, bus.CommandResultTTL)
	ledger.now = func() time.Time { return now }

	if _, _, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID); err != nil {
		t.Fatalf("expected claim to succeed, got %v", err)
	}
	if err := ledger.Complete(context.Background(), testRoomID, testClientID, testCommandID, []byte(`{"type":"command-ack"}`)); err != nil {
		t.Fatalf("expected complete to succeed, got %v", err)
	}

	now = now.Add(bus.CommandResultTTL + time.Second)

	outcome, found, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID)
	if err != nil {
		t.Fatalf("expected the expired outcome to be reclaimable, got %v", err)
	}
	if found {
		t.Fatalf("expected no outcome after the result entry expired, got %s", outcome)
	}
}

func TestLedger_CompleteReplacesPendingTTL(t *testing.T) {
	now := time.Now()
	ledger := New(bus.CommandPendingTTL, bus.CommandResultTTL)
	ledger.now = func() time.Time { return now }

	if _, _, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID); err != nil {
		t.Fatalf("expected claim to succeed, got %v", err)
	}
	payload := []byte(`{"type":"command-ack"}`)
	if err := ledger.Complete(context.Background(), testRoomID, testClientID, testCommandID, payload); err != nil {
		t.Fatalf("expected complete to succeed, got %v", err)
	}

	now = now.Add(bus.CommandPendingTTL + time.Second)

	outcome, found, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID)
	if err != nil {
		t.Fatalf("expected no error replaying the outcome, got %v", err)
	}
	if !found {
		t.Fatal("expected the completed outcome to outlive the pending TTL")
	}
	if string(outcome) != string(payload) {
		t.Fatalf("expected outcome %s, got %s", payload, outcome)
	}
}

func TestLedger_EntriesAreScopedByClientAndCommand(t *testing.T) {
	ledger := New(bus.CommandPendingTTL, bus.CommandResultTTL)

	if _, _, err := ledger.Claim(context.Background(), testRoomID, testClientID, testCommandID); err != nil {
		t.Fatalf("expected claim to succeed, got %v", err)
	}

	outcome, found, err := ledger.Claim(context.Background(), testRoomID, "client-2", testCommandID)
	if err != nil {
		t.Fatalf("expected an independent claim for another client, got %v", err)
	}
	if found {
		t.Fatalf("expected no outcome for another client, got %s", outcome)
	}
}
