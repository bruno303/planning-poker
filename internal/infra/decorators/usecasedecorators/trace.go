package usecasedecorators

import (
	"context"
	"planning-poker/internal/application/planningpoker/usecase"

	"github.com/bruno303/go-toolkit/pkg/trace"
)

type (
	TraceableUseCaseR[In any, Out any] struct {
		inner     usecase.UseCaseR[In, Out]
		traceName string
		spanName  string
	}
	TraceableUseCase[In any] struct {
		inner     usecase.UseCase[In]
		traceName string
		spanName  string
	}
	TraceableUseCaseO[Out any] struct {
		inner     usecase.UseCaseO[Out]
		traceName string
		spanName  string
	}
)

var (
	_ usecase.UseCaseR[any, any] = (*TraceableUseCaseR[any, any])(nil)
	_ usecase.UseCase[any]       = (*TraceableUseCase[any])(nil)
)

func NewTraceableUseCaseR[In any, Out any](inner usecase.UseCaseR[In, Out], traceName, spanName string) *TraceableUseCaseR[In, Out] {
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

func NewTraceableUseCase[In any](inner usecase.UseCase[In], traceName, spanName string) *TraceableUseCase[In] {
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

func NewTraceableUseCaseO[Out any](inner usecase.UseCaseO[Out], traceName, spanName string) *TraceableUseCaseO[Out] {
	return &TraceableUseCaseO[Out]{
		inner:     inner,
		traceName: traceName,
		spanName:  spanName,
	}
}

func (uc *TraceableUseCaseO[Out]) Execute(ctx context.Context) (Out, error) {
	result, err := trace.Trace(ctx, trace.NameConfig(uc.traceName, uc.spanName), func(ctx context.Context) (any, error) {
		return uc.inner.Execute(ctx)
	})

	if err != nil {
		var zero Out
		return zero, err
	}

	return result.(Out), nil
}
