package metric

import (
	"context"
	"reflect"
	"sync/atomic"

	"github.com/bruno303/go-toolkit/pkg/metric"
)

type PlanningPokerMetric struct {
	meter  metric.Meter
	active *activeCounters
}

// activeCounters holds the current values reported as gauges. OTel gauges
// record an absolute value rather than a delta, so the running count has to be
// kept in the process. It is shared by pointer because PlanningPokerMetric is
// passed around by value. The backend runs a single replica per environment
// (helm/values.yaml), so the in-process count is the cluster-wide one; scaling
// out would need a shared counter instead.
type activeCounters struct {
	users atomic.Int64
	rooms atomic.Int64
}

type noopMeter struct{}

const (
	PlanningPokerActiveUsersMetric = "planning_poker_active_users"
	PlanningPokerUsersTotalMetric  = "planning_poker_users_total"
	PlanningPokerActiveRoomsMetric = "planning_poker_active_rooms"
)

func NewPlanningPokerMetric() PlanningPokerMetric {
	return newPlanningPokerMetric(noopMeter{})
}

func NewPlanningPokerMetricWithMeter(meter metric.Meter) PlanningPokerMetric {
	if isNilMeter(meter) {
		return NewPlanningPokerMetric()
	}

	return newPlanningPokerMetric(meter)
}

func newPlanningPokerMetric(meter metric.Meter) PlanningPokerMetric {
	return PlanningPokerMetric{meter: meter, active: &activeCounters{}}
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

func (m PlanningPokerMetric) IncrementActiveUsers(ctx context.Context) {
	m.recordActiveUsers(ctx, m.active.users.Add(1))
}

func (m PlanningPokerMetric) DecrementActiveUsers(ctx context.Context) {
	m.recordActiveUsers(ctx, m.active.users.Add(-1))
}

func (m PlanningPokerMetric) recordActiveUsers(ctx context.Context, value int64) {
	_ = m.meter.AddGauge(ctx, PlanningPokerActiveUsersMetric, "", "", float64(value))
}

func (m PlanningPokerMetric) IncrementUsersTotal(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerUsersTotalMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementUsersTotal(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerUsersTotalMetric, "", "", -1)
}

func (m PlanningPokerMetric) IncrementActiveRooms(ctx context.Context) {
	m.recordActiveRooms(ctx, m.active.rooms.Add(1))
}

func (m PlanningPokerMetric) DecrementActiveRooms(ctx context.Context) {
	m.recordActiveRooms(ctx, m.active.rooms.Add(-1))
}

func (m PlanningPokerMetric) recordActiveRooms(ctx context.Context, value int64) {
	_ = m.meter.AddGauge(ctx, PlanningPokerActiveRoomsMetric, "", "", float64(value))
}
