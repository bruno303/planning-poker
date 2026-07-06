package usecase

import (
	"context"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
)

type (
	CreateClientOutput struct {
		ClientID string
	}
	CreateClientUseCase struct {
		hub    domain.Hub
		logger log.Logger
		metric metric.PlanningPokerMetric
	}
)

var _ UseCaseO[CreateClientOutput] = (*CreateClientUseCase)(nil)

func NewCreateClientUseCase(hub domain.Hub, metric metric.PlanningPokerMetric) CreateClientUseCase {
	return CreateClientUseCase{
		hub:    hub,
		logger: log.NewLogger("usecase.CreateClient"),
		metric: metric,
	}
}

func (uc CreateClientUseCase) Execute(ctx context.Context) (CreateClientOutput, error) {
	clientID := uuid.NewString()
	uc.logger.Info(ctx, "Client created with ID: %s", clientID)

	return CreateClientOutput{ClientID: clientID}, nil
}
