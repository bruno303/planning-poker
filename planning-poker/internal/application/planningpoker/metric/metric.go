package metric

import (
	"context"
	"reflect"

	"github.com/bruno303/go-toolkit/pkg/metric"
)

type PlanningPokerMetric struct {
	meter metric.Meter
}

type noopMeter struct{}

const (
	PlanningPokerActiveUsersMetric = "planning_poker_active_users"
	PlanningPokerUsersTotalMetric  = "planning_poker_users_total"
	PlanningPokerActiveRoomsMetric = "planning_poker_active_rooms"
)

func NewPlanningPokerMetric() PlanningPokerMetric {
	return PlanningPokerMetric{meter: noopMeter{}}
}

func NewPlanningPokerMetricWithMeter(meter metric.Meter) PlanningPokerMetric {
	if isNilMeter(meter) {
		return NewPlanningPokerMetric()
	}

	return PlanningPokerMetric{meter: meter}
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
	_ = m.meter.AddCounter(ctx, PlanningPokerActiveUsersMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementActiveUsers(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerActiveUsersMetric, "", "", -1)
}

func (m PlanningPokerMetric) IncrementUsersTotal(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerUsersTotalMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementUsersTotal(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerUsersTotalMetric, "", "", -1)
}

func (m PlanningPokerMetric) IncrementActiveRoomsCounter(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerActiveRoomsMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementActiveRoomsCounter(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerActiveRoomsMetric, "", "", -1)
}
