package usecasedecorators

import (
	"context"
	"errors"
	"planning-poker/internal/application"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewTraceableUseCaseR(t *testing.T) {
	ctrl := gomock.NewController(t)

	inner := application.NewMockUseCaseR[string, string](ctrl)
	traceName := "test-trace"
	spanName := "test-span"

	decorator := NewTraceableUseCaseR(inner, traceName, spanName)

	if decorator == nil {
		t.Fatal("NewTraceableUseCaseR returned nil")
	}
	if decorator.inner != inner {
		t.Error("inner use case not set correctly")
	}
	if decorator.traceName != traceName {
		t.Errorf("traceName = %v, want %v", decorator.traceName, traceName)
	}
	if decorator.spanName != spanName {
		t.Errorf("spanName = %v, want %v", decorator.spanName, spanName)
	}
}

func TestTraceableUseCaseR_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	expectedResult := "success"
	input := "test-input"
	ctx := context.Background()

	inner := application.NewMockUseCaseR[string, string](ctrl)
	inner.EXPECT().
		Execute(gomock.Any(), input).
		Return(expectedResult, nil)

	decorator := NewTraceableUseCaseR(inner, "test-trace", "test-span")

	result, err := decorator.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != expectedResult {
		t.Errorf("result = %v, want %v", result, expectedResult)
	}
}

func TestTraceableUseCaseR_Execute_Error(t *testing.T) {
	ctrl := gomock.NewController(t)

	expectedError := errors.New("test error")
	ctx := context.Background()

	inner := application.NewMockUseCaseR[string, string](ctrl)
	inner.EXPECT().
		Execute(gomock.Any(), "test-input").
		Return("", expectedError)

	decorator := NewTraceableUseCaseR(inner, "test-trace", "test-span")

	result, err := decorator.Execute(ctx, "test-input")

	if err == nil {
		t.Fatal("Execute should return error")
	}
	if err != expectedError {
		t.Errorf("error = %v, want %v", err, expectedError)
	}
	if result != "" {
		t.Errorf("result should be zero value, got %v", result)
	}
}

func TestTraceableUseCaseR_Execute_WithDifferentTypes(t *testing.T) {
	ctrl := gomock.NewController(t)

	type customInput struct {
		value int
	}
	type customOutput struct {
		result string
	}

	input := customInput{value: 42}
	expectedOutput := customOutput{result: "processed"}
	ctx := context.Background()

	inner := application.NewMockUseCaseR[customInput, customOutput](ctrl)
	inner.EXPECT().
		Execute(gomock.Any(), input).
		Return(expectedOutput, nil)

	decorator := NewTraceableUseCaseR(inner, "test-trace", "test-span")

	result, err := decorator.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.result != "processed" {
		t.Errorf("result.result = %v, want %v", result.result, "processed")
	}
}

func TestNewTraceableUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)

	inner := application.NewMockUseCase[string](ctrl)
	traceName := "test-trace"
	spanName := "test-span"

	decorator := NewTraceableUseCase(inner, traceName, spanName)

	if decorator == nil {
		t.Fatal("NewTraceableUseCase returned nil")
	}
	if decorator.inner != inner {
		t.Error("inner use case not set correctly")
	}
	if decorator.traceName != traceName {
		t.Errorf("traceName = %v, want %v", decorator.traceName, traceName)
	}
	if decorator.spanName != spanName {
		t.Errorf("spanName = %v, want %v", decorator.spanName, spanName)
	}
}

func TestTraceableUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	input := "test-input"
	ctx := context.Background()

	inner := application.NewMockUseCase[string](ctrl)
	inner.EXPECT().
		Execute(gomock.Any(), input).
		Return(nil)

	decorator := NewTraceableUseCase(inner, "test-trace", "test-span")

	err := decorator.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestTraceableUseCase_Execute_Error(t *testing.T) {
	ctrl := gomock.NewController(t)

	expectedError := errors.New("test error")
	ctx := context.Background()

	inner := application.NewMockUseCase[string](ctrl)
	inner.EXPECT().
		Execute(gomock.Any(), "test-input").
		Return(expectedError)

	decorator := NewTraceableUseCase(inner, "test-trace", "test-span")

	err := decorator.Execute(ctx, "test-input")

	if err == nil {
		t.Fatal("Execute should return error")
	}
	if err != expectedError {
		t.Errorf("error = %v, want %v", err, expectedError)
	}
}

func TestTraceableUseCase_Execute_WithDifferentInputType(t *testing.T) {
	ctrl := gomock.NewController(t)

	type customInput struct {
		id   string
		name string
	}

	input := customInput{id: "123", name: "test"}
	ctx := context.Background()

	inner := application.NewMockUseCase[customInput](ctrl)
	inner.EXPECT().
		Execute(gomock.Any(), input).
		Return(nil)

	decorator := NewTraceableUseCase(inner, "test-trace", "test-span")

	err := decorator.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}
