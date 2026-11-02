package bus

import (
	"errors"

	"planning-poker/internal/domain/domainerror"
	"planning-poker/internal/protocol"
)

var (
	ErrInvalidPayload = errors.New("invalid payload")

	domainCodeToWireCode = map[domainerror.Code]protocol.CommandErrorCode{
		domainerror.CodeValidation:   protocol.CommandErrorCodeValidation,
		domainerror.CodeForbidden:    protocol.CommandErrorCodeForbidden,
		domainerror.CodeNotFound:     protocol.CommandErrorCodeNotFound,
		domainerror.CodeConflict:     protocol.CommandErrorCodeConflict,
		domainerror.CodeStaleVersion: protocol.CommandErrorCodeStaleVersion,
	}
)

func newCommandAck(commandID string, roomVersion *uint64) protocol.CommandAck {
	return protocol.CommandAck{
		Type:        protocol.MessageTypeCommandAck,
		CommandID:   commandID,
		RoomVersion: roomVersion,
	}
}

func newCommandError(commandID string, code protocol.CommandErrorCode, message string, roomVersion *uint64) protocol.CommandError {
	return protocol.CommandError{
		Type:        protocol.MessageTypeCommandError,
		CommandID:   commandID,
		Code:        code,
		Message:     message,
		RoomVersion: roomVersion,
	}
}

// commandErrorFor translates a use case failure into the wire error contract.
// The original error is never serialized; only known codes and safe messages
// leave the process.
func commandErrorFor(commandID string, err error) protocol.CommandError {
	if errors.Is(err, ErrInvalidPayload) {
		return newCommandError(commandID, protocol.CommandErrorCodeInvalidPayload, ErrInvalidPayload.Error(), nil)
	}

	domainErr, ok := domainerror.As(err)
	if !ok {
		return newCommandError(commandID, protocol.CommandErrorCodeInternal, "internal error", nil)
	}

	code, ok := domainCodeToWireCode[domainErr.Code()]
	if !ok {
		return newCommandError(commandID, protocol.CommandErrorCodeInternal, "internal error", nil)
	}

	return newCommandError(commandID, code, domainErr.Message(), nil)
}
