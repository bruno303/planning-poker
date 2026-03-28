package usecase

import (
	"context"
	"sync"
	"testing"

	toolkitmetric "github.com/bruno303/go-toolkit/pkg/metric"
	"go.uber.org/mock/gomock"
	appmetric "planning-poker/internal/application/planningpoker/metric"
)

type metricCall struct {
	name        string
	description string
	unit        string
	value       float64
	attributes  []toolkitmetric.Attribute
}

type metricRecorder struct {
	mu    sync.Mutex
	calls []metricCall
}

func newTestPlanningPokerMetric(ctrl *gomock.Controller) (appmetric.PlanningPokerMetric, *metricRecorder) {
	recorder := &metricRecorder{}
	mockMeter := appmetric.NewMockMeter(ctrl)
	mockMeter.EXPECT().
		AddCounter(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(_ context.Context, name, description, unit string, value float64, attributes ...toolkitmetric.Attribute) error {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()

			recorder.calls = append(recorder.calls, metricCall{
				name:        name,
				description: description,
				unit:        unit,
				value:       value,
				attributes:  append([]toolkitmetric.Attribute(nil), attributes...),
			})

			return nil
		})

	return appmetric.NewPlanningPokerMetricWithMeter(mockMeter), recorder
}

func (r *metricRecorder) getCalls() []metricCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]metricCall(nil), r.calls...)
}

func countMetricCalls(calls []metricCall, metricName string) int {
	count := 0
	for _, call := range calls {
		if call.name == metricName {
			count++
		}
	}

	return count
}

func countMetricCallsWithValue(calls []metricCall, metricName string, value float64) int {
	count := 0
	for _, call := range calls {
		if call.name == metricName && call.value == value {
			count++
		}
	}

	return count
}

type expectedMetricCall struct {
	name  string
	value float64
}

func assertMetricCallSequence(t *testing.T, calls []metricCall, expected ...expectedMetricCall) {
	t.Helper()

	if len(calls) != len(expected) {
		t.Fatalf("expected %d metric calls, got %d", len(expected), len(calls))
	}

	for i, want := range expected {
		got := calls[i]
		if got.name != want.name || got.value != want.value {
			t.Fatalf(
				"expected metric call %d to be %q with value %v, got %q with value %v",
				i,
				want.name,
				want.value,
				got.name,
				got.value,
			)
		}
	}
}
