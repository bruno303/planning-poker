package bus

import (
	"context"
	"errors"
	"time"
)

// ErrCommandInProgress signals that another execution currently holds the
// reservation for the given command.
var ErrCommandInProgress = errors.New("command in progress")

const (
	CommandPendingTTL = 30 * time.Second
	CommandResultTTL  = 5 * time.Minute
)

// CommandLedger reserves command IDs and stores their outcomes so retries can be
// answered without re-executing the use case.
//
// Claim contract:
//   - (nil, false, nil): the reservation was acquired and the caller must execute
//     the command and later report the outcome via Complete.
//   - (payload, true, nil): the command already completed; payload is the stored
//     outcome and must be replayed verbatim.
//   - ErrCommandInProgress: another execution currently holds the reservation.
type CommandLedger interface {
	Claim(ctx context.Context, roomID, clientID, commandID string) (outcome []byte, found bool, err error)
	Complete(ctx context.Context, roomID, clientID, commandID string, outcome []byte) error
}
