package domainerror

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewf(t *testing.T) {
	tests := []struct {
		name string
		code Code
		want string
	}{
		{name: "validation", code: CodeValidation, want: "invalid story"},
		{name: "forbidden", code: CodeForbidden, want: "only the room owner can reveal"},
		{name: "not found", code: CodeNotFound, want: "room not found"},
		{name: "conflict", code: CodeConflict, want: "already selected"},
		{name: "stale version", code: CodeStaleVersion, want: "stale room version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Newf(tt.code, "%s", tt.want)

			if got := err.Code(); got != tt.code {
				t.Fatalf("Code() = %q, want %q", got, tt.code)
			}
			if got := err.Message(); got != tt.want {
				t.Fatalf("Message() = %q, want %q", got, tt.want)
			}
			if got := err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewfFormattedMessage(t *testing.T) {
	err := Newf(CodeValidation, "client %s not found in room %s", "client-1", "room-1")

	want := "client client-1 not found in room room-1"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if got := err.Message(); got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}
}

func TestSentinels(t *testing.T) {
	tests := []struct {
		name    string
		err     *Error
		want    Code
		message string
	}{
		{
			name:    "room not found",
			err:     ErrRoomNotFound,
			want:    CodeNotFound,
			message: "room not found",
		},
		{
			name:    "client not found",
			err:     ErrClientNotFound,
			want:    CodeNotFound,
			message: "client not found",
		},
		{
			name:    "last owner",
			err:     ErrLastOwner,
			want:    CodeConflict,
			message: "cannot remove the last owner",
		},
		{
			name:    "stale room version",
			err:     ErrStaleRoomVersion,
			want:    CodeStaleVersion,
			message: "stale room version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.want {
				t.Fatalf("Code() = %q, want %q", got, tt.want)
			}
			if got := tt.err.Message(); got != tt.message {
				t.Fatalf("Message() = %q, want %q", got, tt.message)
			}
			if got := tt.err.Error(); got != tt.message {
				t.Fatalf("Error() = %q, want %q", got, tt.message)
			}
		})
	}
}

func TestAs(t *testing.T) {
	coded := Newf(CodeForbidden, "only the room owner can reveal")

	tests := []struct {
		name     string
		err      error
		want     *Error
		wantOK   bool
		wantCode Code
	}{
		{
			name:     "coded error",
			err:      coded,
			want:     coded,
			wantOK:   true,
			wantCode: CodeForbidden,
		},
		{
			name:     "wrapped coded error",
			err:      fmt.Errorf("update room: %w", coded),
			want:     coded,
			wantOK:   true,
			wantCode: CodeForbidden,
		},
		{
			name:   "plain error",
			err:    errors.New("plain error"),
			want:   nil,
			wantOK: false,
		},
		{
			name:   "nil error",
			err:    nil,
			want:   nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := As(tt.err)

			if ok != tt.wantOK {
				t.Fatalf("As() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("As() error = %v, want %v", got, tt.want)
			}
			if ok && got.Code() != tt.wantCode {
				t.Fatalf("Code() = %q, want %q", got.Code(), tt.wantCode)
			}
		})
	}
}

func TestErrorsIsSentinels(t *testing.T) {
	t.Run("wrapped sentinel matches", func(t *testing.T) {
		err := fmt.Errorf("x: %w", ErrLastOwner)

		if !errors.Is(err, ErrLastOwner) {
			t.Fatal("errors.Is(wrapped ErrLastOwner, ErrLastOwner) = false, want true")
		}
	})

	t.Run("equal message is not an identity match", func(t *testing.T) {
		err := Newf(CodeConflict, "cannot remove the last owner")

		if errors.Is(err, ErrLastOwner) {
			t.Fatal("errors.Is(Newf(CodeConflict, ...), ErrLastOwner) = true, want false")
		}
	})
}

func TestWrapf(t *testing.T) {
	cause := ErrStaleRoomVersion
	err := Wrapf(CodeConflict, cause, "save room %s: %s", "room-1", "conflict")

	if got, want := err.Code(), CodeConflict; got != want {
		t.Fatalf("Code() = %q, want %q", got, want)
	}
	if got, want := err.Message(), "save room room-1: conflict"; got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}
	if got, want := err.Error(), "save room room-1: conflict"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(err, cause) = false, want true")
	}

	coded, ok := As(fmt.Errorf("outer: %w", err))
	if !ok || coded != err {
		t.Fatalf("As(wrapped) = (%v, %v), want (%v, true)", coded, ok, err)
	}
}

func TestWrapfWithoutCause(t *testing.T) {
	err := Wrapf(CodeValidation, nil, "story ID cannot be empty")

	if got := err.Unwrap(); got != nil {
		t.Fatalf("Unwrap() = %v, want nil", got)
	}
	if got := err.Code(); got != CodeValidation {
		t.Fatalf("Code() = %q, want %q", got, CodeValidation)
	}
}
