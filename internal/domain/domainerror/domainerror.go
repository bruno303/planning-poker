package domainerror

import (
	"errors"
	"fmt"
)

type Code string

const (
	CodeValidation   Code = "validation"
	CodeForbidden    Code = "forbidden"
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodeStaleVersion Code = "stale_version"
)

type Error struct {
	code    Code
	message string
	cause   error
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) Code() Code {
	return e.code
}

func (e *Error) Message() string {
	return e.message
}

func Newf(code Code, format string, args ...any) *Error {
	return &Error{
		code:    code,
		message: fmt.Sprintf(format, args...),
	}
}

func Wrapf(code Code, cause error, format string, args ...any) *Error {
	return &Error{
		code:    code,
		message: fmt.Sprintf(format, args...),
		cause:   cause,
	}
}

func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}

	return nil, false
}

var (
	ErrRoomNotFound     = Newf(CodeNotFound, "room not found")
	ErrClientNotFound   = Newf(CodeNotFound, "client not found")
	ErrLastOwner        = Newf(CodeConflict, "cannot remove the last owner")
	ErrStaleRoomVersion = Newf(CodeStaleVersion, "stale room version")
)
