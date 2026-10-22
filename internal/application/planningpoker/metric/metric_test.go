package metric

import (
	"context"
	"testing"

	toolkitmetric "github.com/bruno303/go-toolkit/pkg/metric"
	"go.uber.org/mock/gomock"
)

type recordedCall struct {
	ctx         context.Context
	gauge       bool
	name        string
	description string
	unit        string
	value       float64
	attributes  []toolkitmetric.Attribute
}

func newRecordedMeter(ctrl *gomock.Controller) (*MockMeter, *[]recordedCall) {
	mockMeter := NewMockMeter(ctrl)
	calls := make([]recordedCall, 0)

	record := func(gauge bool) func(context.Context, string, string, string, float64, ...toolkitmetric.Attribute) error {
		return func(ctx context.Context, name, description, unit string, value float64, attributes ...toolkitmetric.Attribute) error {
			calls = append(calls, recordedCall{
				ctx:         ctx,
				gauge:       gauge,
				name:        name,
				description: description,
				unit:        unit,
				value:       value,
				attributes:  append([]toolkitmetric.Attribute(nil), attributes...),
			})

			return nil
		}
	}

	mockMeter.EXPECT().
		AddGauge(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(record(true))
	mockMeter.EXPECT().
		AddCounter(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(record(false))

	return mockMeter, &calls
}

func assertRecordedCall(t *testing.T, call recordedCall, gauge bool, expectedName string, expectedValue float64) {
	t.Helper()

	if call.name != expectedName {
		t.Fatalf("expected metric %q, got %q", expectedName, call.name)
	}
	if call.gauge != gauge {
		t.Fatalf("expected gauge=%v, got %v", gauge, call.gauge)
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
	m.DecrementUsersTotal(ctx)
	m.IncrementActiveRooms(ctx)
	m.DecrementActiveRooms(ctx)
}

func TestNewPlanningPokerMetric_WithInjectedMeter_UsesProvidedMeter(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter := NewMockMeter(ctrl)
	ctx := context.Background()

	mockMeter.EXPECT().
		AddGauge(ctx, PlanningPokerActiveUsersMetric, "", "", 1.0).
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
	m.IncrementActiveRooms(ctx)
	m.DecrementActiveRooms(ctx)
}

func TestPlanningPokerMetric_ActiveMetrics_RecordCurrentValue(t *testing.T) {
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
			name:          "first active user",
			invoke:        m.IncrementActiveUsers,
			expectedName:  PlanningPokerActiveUsersMetric,
			expectedValue: 1,
		},
		{
			name:          "second active user",
			invoke:        m.IncrementActiveUsers,
			expectedName:  PlanningPokerActiveUsersMetric,
			expectedValue: 2,
		},
		{
			name:          "user leaves",
			invoke:        m.DecrementActiveUsers,
			expectedName:  PlanningPokerActiveUsersMetric,
			expectedValue: 1,
		},
		{
			name:          "first active room",
			invoke:        m.IncrementActiveRooms,
			expectedName:  PlanningPokerActiveRoomsMetric,
			expectedValue: 1,
		},
		{
			name:          "room closes",
			invoke:        m.DecrementActiveRooms,
			expectedName:  PlanningPokerActiveRoomsMetric,
			expectedValue: 0,
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

			assertRecordedCall(t, (*calls)[got-1], true, tt.expectedName, tt.expectedValue)
		})
	}
}

func TestPlanningPokerMetric_ActiveCountersAreSharedAcrossCopies(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter, calls := newRecordedMeter(ctrl)
	m := NewPlanningPokerMetricWithMeter(mockMeter)
	copied := m
	ctx := context.Background()

	copied.IncrementActiveUsers(ctx)
	m.IncrementActiveUsers(ctx)

	if len(*calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*calls))
	}
	assertRecordedCall(t, (*calls)[0], true, PlanningPokerActiveUsersMetric, 1)
	assertRecordedCall(t, (*calls)[1], true, PlanningPokerActiveUsersMetric, 2)
}

func TestPlanningPokerMetric_UsersTotal_RecordsCounterCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMeter, calls := newRecordedMeter(ctrl)
	m := NewPlanningPokerMetricWithMeter(mockMeter)
	ctx := context.Background()

	m.IncrementUsersTotal(ctx)
	m.DecrementUsersTotal(ctx)

	if len(*calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*calls))
	}
	assertRecordedCall(t, (*calls)[0], false, PlanningPokerUsersTotalMetric, 1)
	assertRecordedCall(t, (*calls)[1], false, PlanningPokerUsersTotalMetric, -1)
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
