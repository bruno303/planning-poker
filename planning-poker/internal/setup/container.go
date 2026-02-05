package setup

import (
	"context"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/config"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/bus/redis"
	"planning-poker/internal/infra/boundaries/http"
	"planning-poker/internal/infra/decorators/usecasedecorators"
	infralock "planning-poker/internal/infra/lock"

	redislib "github.com/redis/go-redis/v9"
)

type (
	APIContainer struct {
		APIs []http.API
	}
	InfraContainer struct {
		RedisClient *redislib.Client
	}

	Container struct {
		Hub         domain.Hub
		LockManager lock.LockManager
		Usecases    usecase.UseCasesFacade
		API         APIContainer
		Infra       InfraContainer
	}
)

func NewContainer(cfg *config.Config) *Container {
	ctx := context.Background()
	infra, err := newInfraContainer(cfg)
	if err != nil {
		panic("Failed to initialize infrastructure: " + err.Error())
	}

	hub, err := redis.NewRedisHub(ctx, infra.RedisClient)
	if err != nil {
		panic("Failed to initialize Redis hub (ensure Redis is running and accessible): " + err.Error())
	}

	lockManager := infralock.NewRedisLockManager(infra.RedisClient)
	planningPokerMetric := metric.NewPlanningPokerMetric()
	usecases := newUsecases(hub, lockManager, planningPokerMetric)

	return &Container{
		Hub:         hub,
		LockManager: lockManager,
		API:         newAPIContainer(cfg, hub, usecases, hub, infra.RedisClient),
		Usecases:    usecases,
		Infra:       infra,
	}
}

func newInfraContainer(cfg *config.Config) (InfraContainer, error) {
	redisClient, err := NewRedisClient(cfg)
	if err != nil {
		return InfraContainer{}, err
	}

	return InfraContainer{
		RedisClient: redisClient,
	}, nil
}

func newAPIContainer(cfg *config.Config, hub domain.Hub, usecases usecase.UseCasesFacade, adminHub domain.AdminHub, redisClient *redislib.Client) APIContainer {
	healthCheckers := []http.HealthChecker{
		http.NewRedisHealthChecker(redisClient, "redis"),
	}

	return APIContainer{
		APIs: []http.API{
			http.NewWebsocketAPI(hub, usecases, http.WebSocketConfig{
				WriteTimeout: cfg.API.PlanningPoker.WebsocketWriteTimeout,
				ReadTimeout:  cfg.API.PlanningPoker.WebsocketReadTimeout,
				PingInterval: cfg.API.PlanningPoker.WebsocketPingInterval,
			}),
			http.NewGetRoomAPI(hub),
			http.NewCreateRoomAPI(usecases.CreateRoom),
			http.NewHealthcheckAPI(healthCheckers...),
			http.NewGetAllRoomsStateAPI(adminHub, cfg.API.Admin.APIKey),
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
	}
}
