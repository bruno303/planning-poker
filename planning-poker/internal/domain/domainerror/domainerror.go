package domainerror

import "errors"

var (
	ErrRoomNotFound   = errors.New("room not found")
	ErrClientNotFound = errors.New("client not found")
	ErrLastOwner      = errors.New("cannot remove the last owner")
)
