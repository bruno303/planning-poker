package decorator

import (
	"context"

	"github.com/bruno303/go-toolkit/pkg/trace"
)

type (
	TraceableUseCase[In any] struct {
		next      func(ctx context.Context, cmd In) error
		traceName string
		spanName  string
	}
	TraceableUseCaseWithResult[In any, Out any] struct {
		next      func(ctx context.Context, cmd In) (Out, error)
		traceName string
		spanName  string
	}
)

func NewTraceableUseCase[In any](next func(ctx context.Context, cmd In) error, traceName, spanName string) *TraceableUseCase[In] {
	return &TraceableUseCase[In]{
		next:      next,
		traceName: traceName,
		spanName:  spanName,
	}
}

func (uc *TraceableUseCase[In]) Execute(ctx context.Context, cmd In) error {
	_, err := trace.Trace(ctx, trace.NameConfig(uc.traceName, uc.spanName), func(ctx context.Context) (any, error) {
		return nil, uc.next(ctx, cmd)
	})

	return err
}

func NewTraceableUseCaseWithResult[In any, Out any](next func(ctx context.Context, cmd In) (Out, error), traceName, spanName string) *TraceableUseCaseWithResult[In, Out] {
	return &TraceableUseCaseWithResult[In, Out]{
		next:      next,
		traceName: traceName,
		spanName:  spanName,
	}
}

func (uc *TraceableUseCaseWithResult[In, Out]) Execute(ctx context.Context, cmd In) (Out, error) {
	result, err := trace.Trace(ctx, trace.NameConfig(uc.traceName, uc.spanName), func(ctx context.Context) (any, error) {
		return uc.next(ctx, cmd)
	})

	if err != nil {
		var zero Out
		return zero, err
	}

	return result.(Out), nil
}
