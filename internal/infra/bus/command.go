package bus

import (
	"errors"

	"planning-poker/internal/domain/domainerror"
)

const (
	commandAckType   = "command-ack"
	commandErrorType = "command-error"

	wireCodeValidation       = "validation"
	wireCodeForbidden        = "forbidden"
	wireCodeNotFound         = "not_found"
	wireCodeConflict         = "conflict"
	wireCodeStaleVersion     = "stale_version"
	wireCodeInvalidPayload   = "invalid_payload"
	wireCodeUnknownCommand   = "unknown_command"
	wireCodeInvalidCommandID = "invalid_command_id"
	wireCodeInternal         = "internal"
)

var (
	ErrInvalidPayload = errors.New("invalid payload")

	domainCodeToWireCode = map[domainerror.Code]string{
		domainerror.CodeValidation:   wireCodeValidation,
		domainerror.CodeForbidden:    wireCodeForbidden,
		domainerror.CodeNotFound:     wireCodeNotFound,
		domainerror.CodeConflict:     wireCodeConflict,
		domainerror.CodeStaleVersion: wireCodeStaleVersion,
	}
)

type (
	CommandAck struct {
		Type        string  `json:"type"`
		CommandID   string  `json:"commandId"`
		RoomVersion *uint64 `json:"roomVersion,omitempty"`
	}

	CommandError struct {
		Type        string  `json:"type"`
		CommandID   string  `json:"commandId"`
		Code        string  `json:"code"`
		Message     string  `json:"message"`
		RoomVersion *uint64 `json:"roomVersion,omitempty"`
	}
)

func newCommandAck(commandID string, roomVersion *uint64) CommandAck {
	return CommandAck{
		Type:        commandAckType,
		CommandID:   commandID,
		RoomVersion: roomVersion,
	}
}

func newCommandError(commandID, code, message string, roomVersion *uint64) CommandError {
	return CommandError{
		Type:        commandErrorType,
		CommandID:   commandID,
		Code:        code,
		Message:     message,
		RoomVersion: roomVersion,
	}
}

// commandErrorFor translates a use case failure into the wire error contract.
// The original error is never serialized; only known codes and safe messages
// leave the process.
func commandErrorFor(commandID string, err error) CommandError {
	if errors.Is(err, ErrInvalidPayload) {
		return newCommandError(commandID, wireCodeInvalidPayload, ErrInvalidPayload.Error(), nil)
	}

	domainErr, ok := domainerror.As(err)
	if !ok {
		return newCommandError(commandID, wireCodeInternal, "internal error", nil)
	}

	code, ok := domainCodeToWireCode[domainErr.Code()]
	if !ok {
		return newCommandError(commandID, wireCodeInternal, "internal error", nil)
	}

	return newCommandError(commandID, code, domainErr.Message(), nil)
}
