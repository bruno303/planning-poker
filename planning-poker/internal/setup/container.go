package setup

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/config"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/http"
	"planning-poker/internal/infra/boundaries/hub/redis"
	"planning-poker/internal/infra/bus"
	"planning-poker/internal/infra/decorators/usecasedecorators"
	infralock "planning-poker/internal/infra/lock"

	redislib "github.com/redis/go-redis/v9"
)

type (
	APIContainer struct {
		APIs []http.API
	}
	InfraContainer struct {
		RedisClient         *redislib.Client
		WebsocketBusFactory *bus.WebSocketBusFactory
		Hub                 domain.Hub
		AdminHub            domain.AdminHub
		LockManager         lock.LockManager
	}
	ApplicationContainer struct {
		PlanningPokerMetric metric.PlanningPokerMetric
		Usecases            usecase.UseCasesFacade
	}

	Container struct {
		App   *ApplicationContainer
		API   *APIContainer
		Infra *InfraContainer
	}
)

func NewContainer(cfg *config.Config) *Container {
	ctx := context.Background()

	infra := newInfraContainer(ctx, cfg)
	app := newApplicationContainer(infra)
	infra.WebsocketBusFactory = newWebsocketBusFactory(cfg, infra, app)
	api := newAPIContainer(cfg, infra, app)

	return &Container{
		App:   app,
		API:   api,
		Infra: infra,
	}
}

func newInfraContainer(ctx context.Context, cfg *config.Config) *InfraContainer {
	redisClient, err := NewRedisClient(cfg)
	if err != nil {
		panic("Failed to initialize Redis client: " + err.Error())
	}

	hub, err := redis.NewRedisHub(ctx, redisClient)
	if err != nil {
		panic("Failed to initialize Redis hub (ensure Redis is running and accessible): " + err.Error())
	}

	lockManager := infralock.NewRedisLockManager(redisClient)

	return &InfraContainer{
		RedisClient: redisClient,
		Hub:         hub,
		AdminHub:    hub,
		LockManager: lockManager,
	}
}

func newApplicationContainer(infra *InfraContainer) *ApplicationContainer {
	planningPokerMetric := metric.NewPlanningPokerMetric()
	usecases := newUsecases(infra.Hub, infra.LockManager, planningPokerMetric)

	return &ApplicationContainer{
		PlanningPokerMetric: planningPokerMetric,
		Usecases:            usecases,
	}
}

func newAPIContainer(cfg *config.Config, infra *InfraContainer, app *ApplicationContainer) *APIContainer {
	healthCheckers := []http.HealthChecker{
		http.NewRedisHealthChecker(infra.RedisClient, "redis"),
	}

	return &APIContainer{
		APIs: []http.API{
			http.NewWebsocketAPI(app.Usecases, infra.WebsocketBusFactory),
			http.NewGetRoomAPI(infra.Hub),
			http.NewCreateRoomAPI(app.Usecases.CreateRoom),
			http.NewHealthcheckAPI(healthCheckers...),
			http.NewGetAllRoomsStateAPI(infra.AdminHub, cfg.API.Admin.APIKey),
		},
	}
}

func newUsecases(hub domain.Hub, lockManager lock.LockManager, metric metric.PlanningPokerMetric) usecase.UseCasesFacade {
	updateNameUseCase := usecase.NewUpdateNameUseCase(hub)
	voteUseCase := usecase.NewVoteUseCase(hub, lockManager)
	revealUseCase := usecase.NewRevealUseCase(hub, lockManager)
	resetUseCase := usecase.NewResetUseCase(hub, lockManager)
	toggleSpectatorUseCase := usecase.NewToggleSpectatorUseCase(hub, lockManager)
	toggleOwnerUseCase := usecase.NewToggleOwnerUseCase(hub, lockManager)
	updateStoryUseCase := usecase.NewUpdateStoryUseCase(hub, lockManager)
	newVotingUseCase := usecase.NewNewVotingUseCase(hub, lockManager)
	voteAgainUseCase := usecase.NewVoteAgainUseCase(hub, lockManager)
	leaveRoomUseCase := usecase.NewLeaveRoomUseCase(hub, lockManager, metric)
	joinRoomUseCase := usecase.NewJoinRoomUseCase(hub, lockManager, metric)
	createRoomUseCase := usecase.NewCreateRoomUseCase(hub, metric)
	createClientUseCase := usecase.NewCreateClientUseCase(hub, metric)

	return usecase.UseCasesFacade{
		UpdateName:      usecasedecorators.NewTraceableUseCase(updateNameUseCase, "UpdateNameUseCase", "UpdateName"),
		Vote:            usecasedecorators.NewTraceableUseCase(voteUseCase, "VoteUseCase", "Vote"),
		Reveal:          usecasedecorators.NewTraceableUseCase(revealUseCase, "RevealUseCase", "Reveal"),
		Reset:           usecasedecorators.NewTraceableUseCase(resetUseCase, "ResetUseCase", "Reset"),
		ToggleSpectator: usecasedecorators.NewTraceableUseCase(toggleSpectatorUseCase, "ToggleSpectatorUseCase", "ToggleSpectator"),
		ToggleOwner:     usecasedecorators.NewTraceableUseCase(toggleOwnerUseCase, "ToggleOwnerUseCase", "ToggleOwner"),
		UpdateStory:     usecasedecorators.NewTraceableUseCase(updateStoryUseCase, "UpdateStoryUseCase", "UpdateStory"),
		NewVoting:       usecasedecorators.NewTraceableUseCase(newVotingUseCase, "NewVotingUseCase", "NewVoting"),
		VoteAgain:       usecasedecorators.NewTraceableUseCase(voteAgainUseCase, "VoteAgainUseCase", "VoteAgain"),
		LeaveRoom:       usecasedecorators.NewTraceableUseCase(leaveRoomUseCase, "LeaveRoomUseCase", "LeaveRoom"),
		JoinRoom:        usecasedecorators.NewTraceableUseCaseR(joinRoomUseCase, "JoinRoomUseCase", "JoinRoom"),
		CreateRoom:      usecasedecorators.NewTraceableUseCaseR(createRoomUseCase, "CreateRoomUseCase", "CreateRoom"),
		CreateClient:    usecasedecorators.NewTraceableUseCaseO(createClientUseCase, "CreateClientUseCase", "CreateClient"),
	}
}

func newWebsocketBusFactory(cfg *config.Config, infra *InfraContainer, app *ApplicationContainer) *bus.WebSocketBusFactory {
	return bus.NewWebSocketBusFactory(infra.Hub, app.Usecases, bus.WebSocketConfig{
		WriteTimeout: cfg.API.PlanningPoker.WebsocketWriteTimeout,
		ReadTimeout:  cfg.API.PlanningPoker.WebsocketReadTimeout,
		PingInterval: cfg.API.PlanningPoker.WebsocketPingInterval,
	})
}
