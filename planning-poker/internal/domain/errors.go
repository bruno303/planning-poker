package domain

import "planning-poker/internal/domain/domainerror"

var (
	ErrRoomNotFound   = domainerror.ErrRoomNotFound
	ErrClientNotFound = domainerror.ErrClientNotFound
	ErrLastOwner      = domainerror.ErrLastOwner
)
