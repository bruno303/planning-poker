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

func TestNewPlanningPokerMetric_WithoutInjectedMeter_UsesNoopMeter(t *testing.T) {
	m := NewPlanningPokerMetric()
	ctx := context.Background()

	m.IncrementActiveUsers(ctx)
	m.DecrementActiveUsers(ctx)
	m.IncrementUsersTotal(ctx)
	m.DecrementUsersTotal(ctx)
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
	m.DecrementUsersTotal(ctx)
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
			name:          "decrement users total",
			invoke:        m.DecrementUsersTotal,
			expectedName:  PlanningPokerUsersTotalMetric,
			expectedValue: -1,
		},
		{
			name:          "increment active rooms",
			invoke:        m.IncrementActiveRoomsCounter,
			expectedName:  PlanningPokerActiveRoomsMetric,
			expectedValue: 1,
		},
		{
			name:          "decrement active rooms",
			invoke:        m.DecrementActiveRoomsCounter,
			expectedName:  PlanningPokerActiveRoomsMetric,
			expectedValue: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(*calls)
			tt.invoke(ctx)

			got := len(*calls)
			if got != before+1 {
				t.Fatalf("expected one new call, got %d total calls", got)
			}

			call := (*calls)[got-1]
			if call.name != tt.expectedName {
				t.Fatalf("expected metric %q, got %q", tt.expectedName, call.name)
			}
			if call.value != tt.expectedValue {
				t.Fatalf("expected value %v, got %v", tt.expectedValue, call.value)
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
