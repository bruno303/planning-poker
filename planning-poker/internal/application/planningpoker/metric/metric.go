package metric

import (
	"context"

	"github.com/bruno303/go-toolkit/pkg/metric"
)

type PlanningPokerMetric struct {
	meter metric.Meter
}

const (
	PlanningPokerActiveUsersMetric = "planning_poker_active_users"
	PlanningPokerUsersTotalMetric  = "planning_poker_users_total"
	PlanningPokerActiveRoomsMetric = "planning_poker_active_rooms"
)

func NewPlanningPokerMetric() PlanningPokerMetric {
	return PlanningPokerMetric{
		meter: metric.GetMeter(),
	}
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

func (m PlanningPokerMetric) IncrementActiveRoomsCounter(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerActiveRoomsMetric, "", "", 1)
}

func (m PlanningPokerMetric) DecrementActiveRoomsCounter(ctx context.Context) {
	_ = m.meter.AddCounter(ctx, PlanningPokerActiveRoomsMetric, "", "", -1)
}
