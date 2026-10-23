package metric

import (
	"context"
	"testing"

	toolkitmetric "github.com/bruno303/go-toolkit/pkg/metric"
	"go.uber.org/mock/gomock"
)

type recordedCounterCall struct {
	ctx         context.Context
	name        string
	description string
	unit        string
	value       float64
	attributes  []toolkitmetric.Attribute
}

func newRecordedMeter(ctrl *gomock.Controller) (*MockMeter, *[]recordedCounterCall) {
	mockMeter := NewMockMeter(ctrl)
	calls := make([]recordedCounterCall, 0)

	mockMeter.EXPECT().
		AddCounter(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(ctx context.Context, name, description, unit string, value float64, attributes ...toolkitmetric.Attribute) error {
			calls = append(calls, recordedCounterCall{
				ctx:         ctx,
				name:        name,
				description: description,
				unit:        unit,
				value:       value,
				attributes:  append([]toolkitmetric.Attribute(nil), attributes...),
			})

			return nil
		})

	return mockMeter, &calls
}

func assertRecordedCounterCall(t *testing.T, call recordedCounterCall, expectedName string, expectedValue float64) {
	t.Helper()

	if call.name != expectedName {
		t.Fatalf("expected metric %q, got %q", expectedName, call.name)
	}
	if call.value != expectedValue {
		t.Fatalf("expected value %v, got %v", expectedValue, call.value)
	}
	if call.description != "" {
		t.Fatalf("expected empty description, got %q", call.description)
	}
	if call.unit != "" {
		t.Fatalf("expected empty unit, got %q", call.unit)
	}
	if len(call.attributes) != 0 {
		t.Fatalf("expected no attributes, got %d", len(call.attributes))
	}
}

func TestNewPlanningPokerMetric_WithoutInjectedMeter_UsesNoopMeter(t *testing.T) {
	m := NewPlanningPokerMetric()
	ctx := context.Background()

	m.IncrementActiveUsers(ctx)
	m.DecrementActiveUsers(ctx)
	m.IncrementUsersTotal(ctx)
	m.IncrementActiveRoomsCounter(ctx)
	m.DecrementActiveRoomsCounter(ctx)
}

func TestNewPlanningPokerMetric_WithInjectedMeter_UsesProvidedMeter(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter := NewMockMeter(ctrl)
	ctx := context.Background()

	mockMeter.EXPECT().
		AddCounter(ctx, PlanningPokerActiveUsersMetric, "", "", 1.0).
		Return(nil)

	m := NewPlanningPokerMetricWithMeter(mockMeter)
	m.IncrementActiveUsers(ctx)
}

func TestNewPlanningPokerMetric_WithTypedNilMeter_FallsBackToNoopMeter(t *testing.T) {
	var typedNilMeter *MockMeter
	m := NewPlanningPokerMetricWithMeter(typedNilMeter)
	ctx := context.Background()

	m.IncrementActiveUsers(ctx)
	m.DecrementActiveUsers(ctx)
	m.IncrementUsersTotal(ctx)
	m.IncrementActiveRoomsCounter(ctx)
	m.DecrementActiveRoomsCounter(ctx)
}

func TestPlanningPokerMetric_CounterMethods_RecordExpectedCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter, calls := newRecordedMeter(ctrl)
	m := NewPlanningPokerMetricWithMeter(mockMeter)
	ctx := context.Background()

	tests := []struct {
		name          string
		setup         func(context.Context)
		invoke        func(context.Context)
		expectedName  string
		expectedValue float64
	}{
		{
			name:          "increment active users",
			invoke:        m.IncrementActiveUsers,
			expectedName:  PlanningPokerActiveUsersMetric,
			expectedValue: 1,
		},
		{
			name:          "decrement active users",
			setup:         m.IncrementActiveUsers,
			invoke:        m.DecrementActiveUsers,
			expectedName:  PlanningPokerActiveUsersMetric,
			expectedValue: -1,
		},
		{
			name:          "increment users total",
			invoke:        m.IncrementUsersTotal,
			expectedName:  PlanningPokerUsersTotalMetric,
			expectedValue: 1,
		},
		{
			name:          "increment active rooms",
			invoke:        m.IncrementActiveRoomsCounter,
			expectedName:  PlanningPokerActiveRoomsMetric,
			expectedValue: 1,
		},
		{
			name:          "decrement active rooms",
			setup:         m.IncrementActiveRoomsCounter,
			invoke:        m.DecrementActiveRoomsCounter,
			expectedName:  PlanningPokerActiveRoomsMetric,
			expectedValue: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(ctx)
			}

			before := len(*calls)
			tt.invoke(ctx)

			got := len(*calls)
			if got != before+1 {
				t.Fatalf("expected one new call, got %d total calls", got)
			}

			assertRecordedCounterCall(t, (*calls)[got-1], tt.expectedName, tt.expectedValue)
		})
	}
}

func TestPlanningPokerMetric_PropagatesContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter, calls := newRecordedMeter(ctrl)
	m := NewPlanningPokerMetricWithMeter(mockMeter)

	type contextKey string
	const key contextKey = "request-id"
	ctx := context.WithValue(context.Background(), key, "abc-123")

	m.IncrementActiveUsers(ctx)

	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}
	if got := (*calls)[0].ctx.Value(key); got != "abc-123" {
		t.Fatalf("expected propagated context value %q, got %#v", "abc-123", got)
	}
}

func TestPlanningPokerMetric_DecrementIsFlooredAtZero(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		increment  func(PlanningPokerMetric, context.Context)
		decrement  func(PlanningPokerMetric, context.Context)
	}{
		{
			name:       "active users",
			metricName: PlanningPokerActiveUsersMetric,
			increment:  PlanningPokerMetric.IncrementActiveUsers,
			decrement:  PlanningPokerMetric.DecrementActiveUsers,
		},
		{
			name:       "active rooms",
			metricName: PlanningPokerActiveRoomsMetric,
			increment:  PlanningPokerMetric.IncrementActiveRoomsCounter,
			decrement:  PlanningPokerMetric.DecrementActiveRoomsCounter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+": decrement without increment emits nothing", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMeter, calls := newRecordedMeter(ctrl)
			m := NewPlanningPokerMetricWithMeter(mockMeter)
			ctx := context.Background()

			tt.decrement(m, ctx)

			if len(*calls) != 0 {
				t.Fatalf("expected no metric calls, got %d", len(*calls))
			}
		})

		t.Run(tt.name+": increment then decrement emits one positive and one negative", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMeter, calls := newRecordedMeter(ctrl)
			m := NewPlanningPokerMetricWithMeter(mockMeter)
			ctx := context.Background()

			tt.increment(m, ctx)
			tt.decrement(m, ctx)

			if len(*calls) != 2 {
				t.Fatalf("expected 2 metric calls, got %d", len(*calls))
			}
			assertRecordedCounterCall(t, (*calls)[0], tt.metricName, 1)
			assertRecordedCounterCall(t, (*calls)[1], tt.metricName, -1)
		})

		t.Run(tt.name+": repeated decrements emit exactly one negative", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMeter, calls := newRecordedMeter(ctrl)
			m := NewPlanningPokerMetricWithMeter(mockMeter)
			ctx := context.Background()

			tt.increment(m, ctx)
			tt.decrement(m, ctx)
			tt.decrement(m, ctx)

			if len(*calls) != 2 {
				t.Fatalf("expected 2 metric calls, got %d", len(*calls))
			}
			assertRecordedCounterCall(t, (*calls)[0], tt.metricName, 1)
			assertRecordedCounterCall(t, (*calls)[1], tt.metricName, -1)
		})
	}
}

func TestPlanningPokerMetric_ValueCopiesShareCounterState(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter, calls := newRecordedMeter(ctrl)
	original := NewPlanningPokerMetricWithMeter(mockMeter)
	ctx := context.Background()

	copied := original

	original.IncrementActiveUsers(ctx)
	original.IncrementActiveRoomsCounter(ctx)
	copied.DecrementActiveUsers(ctx)
	copied.DecrementActiveRoomsCounter(ctx)

	if len(*calls) != 4 {
		t.Fatalf("expected 4 metric calls, got %d", len(*calls))
	}
	assertRecordedCounterCall(t, (*calls)[0], PlanningPokerActiveUsersMetric, 1)
	assertRecordedCounterCall(t, (*calls)[1], PlanningPokerActiveRoomsMetric, 1)
	assertRecordedCounterCall(t, (*calls)[2], PlanningPokerActiveUsersMetric, -1)
	assertRecordedCounterCall(t, (*calls)[3], PlanningPokerActiveRoomsMetric, -1)
}

func TestPlanningPokerMetric_MetricNames(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{name: "active users", constant: PlanningPokerActiveUsersMetric, expected: "planning_poker_active_users"},
		{name: "users total", constant: PlanningPokerUsersTotalMetric, expected: "planning_poker_users_total"},
		{name: "active rooms", constant: PlanningPokerActiveRoomsMetric, expected: "planning_poker_active_rooms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}
