package metric

import (
	"context"
	"reflect"
	"sync/atomic"

	"github.com/bruno303/go-toolkit/pkg/metric"
)

type PlanningPokerMetric struct {
	meter    metric.Meter
	counters *metricCounters
}

// metricCounters is shared through a pointer: PlanningPokerMetric is passed by
// value into use cases, so every copy must observe the same totals.
type metricCounters struct {
	activeUsers atomic.Int64
	activeRooms atomic.Int64
}

type noopMeter struct{}

const (
	PlanningPokerActiveUsersMetric = "planning_poker_active_users"
	PlanningPokerUsersTotalMetric  = "planning_poker_users_total"
	PlanningPokerActiveRoomsMetric = "planning_poker_active_rooms"
)

func NewPlanningPokerMetric() PlanningPokerMetric {
	return PlanningPokerMetric{meter: noopMeter{}, counters: &metricCounters{}}
}

func NewPlanningPokerMetricWithMeter(meter metric.Meter) PlanningPokerMetric {
	if isNilMeter(meter) {
		return NewPlanningPokerMetric()
	}

	return PlanningPokerMetric{meter: meter, counters: &metricCounters{}}
}

func isNilMeter(meter metric.Meter) bool {
	if meter == nil {
		return true
	}

	value := reflect.ValueOf(meter)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (noopMeter) AddCounter(context.Context, string, string, string, float64, ...metric.Attribute) error {
	return nil
}

func (noopMeter) AddGauge(context.Context, string, string, string, float64, ...metric.Attribute) error {
	return nil
}

func decrementIfPositive(counter *atomic.Int64) bool {
	for {
		current := counter.Load()
		if current <= 0 {
			return false
		}

		if counter.CompareAndSwap(current, current-1) {
			return true
		}
	}
}

func (m PlanningPokerMetric) IncrementActiveUsers(ctx context.Context) {
	m.counters.activeUsers.Add(1)

	_ = m.meter.AddCounter(ctx, PlanningPokerActiveUsersMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementActiveUsers(ctx context.Context) {
	if !decrementIfPositive(&m.counters.activeUsers) {
		return
	}

	_ = m.meter.AddCounter(ctx, PlanningPokerActiveUsersMetric, "", "", -1)
}

func (m PlanningPokerMetric) IncrementUsersTotal(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerUsersTotalMetric, "", "", 1)
}

func (m PlanningPokerMetric) IncrementActiveRoomsCounter(ctx context.Context) {
	m.counters.activeRooms.Add(1)

	_ = m.meter.AddCounter(ctx, PlanningPokerActiveRoomsMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementActiveRoomsCounter(ctx context.Context) {
	if !decrementIfPositive(&m.counters.activeRooms) {
		return
	}

	_ = m.meter.AddCounter(ctx, PlanningPokerActiveRoomsMetric, "", "", -1)
}
