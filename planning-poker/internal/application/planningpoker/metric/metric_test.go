package metric

import (
	"context"
	"sync"
	"testing"

	"github.com/bruno303/go-toolkit/pkg/metric"
)

// mockMeter is a mock implementation of metric.Meter for testing
type mockMeter struct {
	mu      sync.Mutex
	calls   []meterCall
	callErr error
}

type meterCall struct {
	name        string
	description string
	unit        string
	value       float64
	attributes  []metric.Attribute
}

func (m *mockMeter) AddCounter(ctx context.Context, name, description, unit string, value float64, attributes ...metric.Attribute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, meterCall{
		name:        name,
		description: description,
		unit:        unit,
		value:       value,
		attributes:  attributes,
	})
	return m.callErr
}

func (m *mockMeter) AddGauge(ctx context.Context, name, description, unit string, value float64, attributes ...metric.Attribute) error {
	// Not used in PlanningPokerMetric but required to implement the interface
	return nil
}

func (m *mockMeter) getCalls() []meterCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]meterCall{}, m.calls...)
}

// Helper to create PlanningPokerMetric with mock meter
func newTestMetric(mock metric.Meter) PlanningPokerMetric {
	return PlanningPokerMetric{
		meter: mock,
	}
}

func TestNewPlanningPokerMetric(t *testing.T) {
	m := NewPlanningPokerMetric()

	if m.meter == nil {
		t.Error("NewPlanningPokerMetric() meter is nil")
	}
}

func TestPlanningPokerMetric_IncrementActiveUsers(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	m.IncrementActiveUsers(ctx)

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("IncrementActiveUsers() expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.name != PlanningPokerActiveUsersMetric {
		t.Errorf("IncrementActiveUsers() name = %v, want %v", call.name, PlanningPokerActiveUsersMetric)
	}
	if call.value != 1 {
		t.Errorf("IncrementActiveUsers() value = %v, want 1", call.value)
	}
}

func TestPlanningPokerMetric_IncrementActiveUsers_Multiple(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	// Increment multiple times
	for i := 0; i < 5; i++ {
		m.IncrementActiveUsers(ctx)
	}

	calls := mock.getCalls()
	if len(calls) != 5 {
		t.Fatalf("IncrementActiveUsers() expected 5 calls, got %d", len(calls))
	}

	for i, call := range calls {
		if call.name != PlanningPokerActiveUsersMetric {
			t.Errorf("Call %d: name = %v, want %v", i, call.name, PlanningPokerActiveUsersMetric)
		}
		if call.value != 1 {
			t.Errorf("Call %d: value = %v, want 1", i, call.value)
		}
	}
}

func TestPlanningPokerMetric_DecrementActiveUsers(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	m.DecrementActiveUsers(ctx)

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("DecrementActiveUsers() expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.name != PlanningPokerActiveUsersMetric {
		t.Errorf("DecrementActiveUsers() name = %v, want %v", call.name, PlanningPokerActiveUsersMetric)
	}
	if call.value != -1 {
		t.Errorf("DecrementActiveUsers() value = %v, want -1", call.value)
	}
}

func TestPlanningPokerMetric_IncrementUsersTotal(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	m.IncrementUsersTotal(ctx)

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("IncrementUsersTotal() expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.name != PlanningPokerUsersTotalMetric {
		t.Errorf("IncrementUsersTotal() name = %v, want %v", call.name, PlanningPokerUsersTotalMetric)
	}
	if call.value != 1 {
		t.Errorf("IncrementUsersTotal() value = %v, want 1", call.value)
	}
}

func TestPlanningPokerMetric_IncrementActiveRoomsCounter(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	m.IncrementActiveRoomsCounter(ctx)

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("IncrementActiveRoomsCounter() expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.name != PlanningPokerActiveRoomsMetric {
		t.Errorf("IncrementActiveRoomsCounter() name = %v, want %v", call.name, PlanningPokerActiveRoomsMetric)
	}
	if call.value != 1 {
		t.Errorf("IncrementActiveRoomsCounter() value = %v, want 1", call.value)
	}
}

func TestPlanningPokerMetric_DecrementActiveRoomsCounter(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	m.DecrementActiveRoomsCounter(ctx)

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("DecrementActiveRoomsCounter() expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.name != PlanningPokerActiveRoomsMetric {
		t.Errorf("DecrementActiveRoomsCounter() name = %v, want %v", call.name, PlanningPokerActiveRoomsMetric)
	}
	if call.value != -1 {
		t.Errorf("DecrementActiveRoomsCounter() value = %v, want -1", call.value)
	}
}

func TestPlanningPokerMetric_UserLifecycle(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	// Simulate user lifecycle
	m.IncrementUsersTotal(ctx)
	m.IncrementActiveUsers(ctx)
	m.DecrementActiveUsers(ctx)

	calls := mock.getCalls()
	if len(calls) != 3 {
		t.Fatalf("Expected 3 calls, got %d", len(calls))
	}

	// Check first call - IncrementUsersTotal
	if calls[0].name != PlanningPokerUsersTotalMetric {
		t.Errorf("First call name = %v, want %v", calls[0].name, PlanningPokerUsersTotalMetric)
	}
	if calls[0].value != 1 {
		t.Errorf("First call value = %v, want 1", calls[0].value)
	}

	// Check second call - IncrementActiveUsers
	if calls[1].name != PlanningPokerActiveUsersMetric {
		t.Errorf("Second call name = %v, want %v", calls[1].name, PlanningPokerActiveUsersMetric)
	}
	if calls[1].value != 1 {
		t.Errorf("Second call value = %v, want 1", calls[1].value)
	}

	// Check third call - DecrementActiveUsers
	if calls[2].name != PlanningPokerActiveUsersMetric {
		t.Errorf("Third call name = %v, want %v", calls[2].name, PlanningPokerActiveUsersMetric)
	}
	if calls[2].value != -1 {
		t.Errorf("Third call value = %v, want -1", calls[2].value)
	}
}

func TestPlanningPokerMetric_RoomLifecycle(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	// Simulate room lifecycle
	m.IncrementActiveRoomsCounter(ctx)
	m.DecrementActiveRoomsCounter(ctx)

	calls := mock.getCalls()
	if len(calls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(calls))
	}

	// Check first call - IncrementActiveRoomsCounter
	if calls[0].name != PlanningPokerActiveRoomsMetric {
		t.Errorf("First call name = %v, want %v", calls[0].name, PlanningPokerActiveRoomsMetric)
	}
	if calls[0].value != 1 {
		t.Errorf("First call value = %v, want 1", calls[0].value)
	}

	// Check second call - DecrementActiveRoomsCounter
	if calls[1].name != PlanningPokerActiveRoomsMetric {
		t.Errorf("Second call name = %v, want %v", calls[1].name, PlanningPokerActiveRoomsMetric)
	}
	if calls[1].value != -1 {
		t.Errorf("Second call value = %v, want -1", calls[1].value)
	}
}

func TestPlanningPokerMetric_AllMethods(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	// Call all methods
	m.IncrementActiveUsers(ctx)
	m.DecrementActiveUsers(ctx)
	m.IncrementUsersTotal(ctx)
	m.IncrementActiveRoomsCounter(ctx)
	m.DecrementActiveRoomsCounter(ctx)

	calls := mock.getCalls()
	if len(calls) != 5 {
		t.Fatalf("Expected 5 calls, got %d", len(calls))
	}

	expectedCalls := []struct {
		name  string
		value float64
	}{
		{PlanningPokerActiveUsersMetric, 1},
		{PlanningPokerActiveUsersMetric, -1},
		{PlanningPokerUsersTotalMetric, 1},
		{PlanningPokerActiveRoomsMetric, 1},
		{PlanningPokerActiveRoomsMetric, -1},
	}

	for i, expected := range expectedCalls {
		if calls[i].name != expected.name {
			t.Errorf("Call %d: name = %v, want %v", i, calls[i].name, expected.name)
		}
		if calls[i].value != expected.value {
			t.Errorf("Call %d: value = %v, want %v", i, calls[i].value, expected.value)
		}
	}
}

func TestPlanningPokerMetric_MetricNames(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "active users metric name",
			constant: PlanningPokerActiveUsersMetric,
			expected: "planning_poker_active_users",
		},
		{
			name:     "users total metric name",
			constant: PlanningPokerUsersTotalMetric,
			expected: "planning_poker_users_total",
		},
		{
			name:     "active rooms metric name",
			constant: PlanningPokerActiveRoomsMetric,
			expected: "planning_poker_active_rooms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Metric name = %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}

func TestPlanningPokerMetric_ConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	// Simulate concurrent calls
	var wg sync.WaitGroup
	iterations := 10

	wg.Add(5 * iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			m.IncrementActiveUsers(ctx)
		}()
		go func() {
			defer wg.Done()
			m.DecrementActiveUsers(ctx)
		}()
		go func() {
			defer wg.Done()
			m.IncrementUsersTotal(ctx)
		}()
		go func() {
			defer wg.Done()
			m.IncrementActiveRoomsCounter(ctx)
		}()
		go func() {
			defer wg.Done()
			m.DecrementActiveRoomsCounter(ctx)
		}()
	}

	wg.Wait()

	calls := mock.getCalls()
	expectedCalls := 5 * iterations
	if len(calls) != expectedCalls {
		t.Errorf("Expected %d calls, got %d", expectedCalls, len(calls))
	}
}

func TestPlanningPokerMetric_ContextPropagation(t *testing.T) {
	type contextKey string
	key := contextKey("test-key")
	value := "test-value"

	ctx := context.WithValue(context.Background(), key, value)

	mock := &mockMeter{}
	m := newTestMetric(mock)

	// Call with context
	m.IncrementActiveUsers(ctx)

	// Verify the mock was called (context is passed through)
	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
}

func TestPlanningPokerMetric_EmptyDescriptionAndUnit(t *testing.T) {
	ctx := context.Background()
	mock := &mockMeter{}
	m := newTestMetric(mock)

	m.IncrementActiveUsers(ctx)

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.description != "" {
		t.Errorf("Expected empty description, got %v", call.description)
	}
	if call.unit != "" {
		t.Errorf("Expected empty unit, got %v", call.unit)
	}
}
