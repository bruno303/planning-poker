package application

import "context"

//go:generate go tool mockgen -destination mocks.go -typed -package application . UseCase,UseCaseR

type (
	// Represents a Use Case that does not return a result.
	UseCase[In any] interface {
		Execute(ctx context.Context, cmd In) error
	}

	// Represents a Use Case that returns a result.
	UseCaseR[In any, Out any] interface {
		Execute(ctx context.Context, cmd In) (Out, error)
	}
)
