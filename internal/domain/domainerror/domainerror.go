package domainerror

import "errors"

var (
	ErrRoomNotFound     = errors.New("room not found")
	ErrClientNotFound   = errors.New("client not found")
	ErrLastOwner        = errors.New("cannot remove the last owner")
	ErrStaleRoomVersion = errors.New("stale room version")
	ErrUnknownDeck      = errors.New("unknown deck preset")
	ErrVoteNotInDeck    = errors.New("vote is not a card of the room deck")
)
