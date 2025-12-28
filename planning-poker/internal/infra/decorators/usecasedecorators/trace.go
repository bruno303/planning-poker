package usecasedecorators

import (
	"context"
	"planning-poker/internal/application"

	"github.com/bruno303/go-toolkit/pkg/trace"
)

type (
	TraceableUseCaseR[In any, Out any] struct {
		inner     application.UseCaseR[In, Out]
		traceName string
		spanName  string
	}
	TraceableUseCase[In any] struct {
		inner     application.UseCase[In]
		traceName string
		spanName  string
	}
)

var (
	_ application.UseCaseR[any, any] = (*TraceableUseCaseR[any, any])(nil)
	_ application.UseCase[any]       = (*TraceableUseCase[any])(nil)
)

func NewTraceableUseCaseR[In any, Out any](inner application.UseCaseR[In, Out], traceName, spanName string) *TraceableUseCaseR[In, Out] {
	return &TraceableUseCaseR[In, Out]{
		inner:     inner,
		traceName: traceName,
		spanName:  spanName,
	}
}

func (uc *TraceableUseCaseR[In, Out]) Execute(ctx context.Context, cmd In) (Out, error) {
	result, err := trace.Trace(ctx, trace.NameConfig(uc.traceName, uc.spanName), func(ctx context.Context) (any, error) {
		return uc.inner.Execute(ctx, cmd)
	})

	if err != nil {
		var zero Out
		return zero, err
	}

	return result.(Out), nil
}

func NewTraceableUseCase[In any](inner application.UseCase[In], traceName, spanName string) *TraceableUseCase[In] {
	return &TraceableUseCase[In]{
		inner:     inner,
		traceName: traceName,
		spanName:  spanName,
	}
}

func (uc *TraceableUseCase[In]) Execute(ctx context.Context, cmd In) error {
	_, err := trace.Trace(ctx, trace.NameConfig(uc.traceName, uc.spanName), func(ctx context.Context) (any, error) {
		return nil, uc.inner.Execute(ctx, cmd)
	})

	return err
}
