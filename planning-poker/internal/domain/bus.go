package domain

import "context"

type Bus interface {
	Close() error
	Detach()
	Send(ctx context.Context, message any) error
	Listen(ctx context.Context)
	RoomID() string
}
