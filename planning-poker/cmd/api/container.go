package main

import (
	"context"
	applock "planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/config"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"planning-poker/internal/infra/boundaries/http"
	"planning-poker/internal/infra/lock"
	"planning-poker/internal/infra/trace/decorator"
)

type (
	APIContainer struct {
		APIs []http.API
	}

	Container struct {
		Hub         *inmemory.InMemoryHub
		LockManager *lock.InMemoryLockManager
		Usecases    usecase.UseCases
		API         APIContainer
	}
)

func NewContainer(cfg *config.Config) *Container {
	hub := inmemory.NewHub()
	lockManager := lock.NewInMemoryLockManager()
	usecases := newUsecases(hub, lockManager)

	return &Container{
		Hub:         hub,
		LockManager: lockManager,
		API:         newAPIContainer(cfg, hub, usecases, hub),
		Usecases:    usecases,
	}
}

func newAPIContainer(cfg *config.Config, hub domain.Hub, usecases usecase.UseCases, adminHub domain.AdminHub) APIContainer {
	return APIContainer{
		APIs: []http.API{
			http.NewWebsocketAPI(hub, usecases, http.WebSocketConfig{
				WriteTimeout: cfg.API.PlanningPoker.WebsocketWriteTimeout,
				ReadTimeout:  cfg.API.PlanningPoker.WebsocketReadTimeout,
				PingInterval: cfg.API.PlanningPoker.WebsocketPingInterval,
			}),
			http.NewGetRoomAPI(hub),
			http.NewCreateRoomAPI(hub),
			http.NewHealthcheckAPI(),
			http.NewGetAllRoomsStateAPI(adminHub, cfg.API.Admin.APIKey),
		},
	}
}

func newUsecases(hub domain.Hub, lockManager applock.LockManager) usecase.UseCases {
	updateNameUseCase := usecase.NewUpdateNameUseCase(hub)
	voteUseCase := usecase.NewVoteUseCase(hub, lockManager)
	revealUseCase := usecase.NewRevealUseCase(hub, lockManager)
	resetUseCase := usecase.NewResetUseCase(hub, lockManager)
	toggleSpectatorUseCase := usecase.NewToggleSpectatorUseCase(hub, lockManager)
	toggleOwnerUseCase := usecase.NewToggleOwnerUseCase(hub, lockManager)
	updateStoryUseCase := usecase.NewUpdateStoryUseCase(hub, lockManager)
	newVotingUseCase := usecase.NewNewVotingUseCase(hub, lockManager)
	voteAgainUseCase := usecase.NewVoteAgainUseCase(hub, lockManager)
	leaveRoomUseCase := usecase.NewLeaveRoomUseCase(hub, lockManager)
	joinRoomUseCase := usecase.NewJoinRoomUseCase(hub, lockManager)

	return usecase.UseCases{
		UpdateName: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.UpdateNameCommand) error {
			return updateNameUseCase.Execute(ctx, cmd)
		}, "UpdateNameUseCase", "UpdateName"),
		Vote: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.VoteCommand) error {
			return voteUseCase.Execute(ctx, cmd)
		}, "VoteUseCase", "Vote"),
		Reveal: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.RevealCommand) error {
			return revealUseCase.Execute(ctx, cmd)
		}, "RevealUseCase", "Reveal"),
		Reset: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.ResetCommand) error {
			return resetUseCase.Execute(ctx, cmd)
		}, "ResetUseCase", "Reset"),
		ToggleSpectator: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.ToggleSpectatorCommand) error {
			return toggleSpectatorUseCase.Execute(ctx, cmd)
		}, "ToggleSpectatorUseCase", "ToggleSpectator"),
		ToggleOwner: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.ToggleOwnerCommand) error {
			return toggleOwnerUseCase.Execute(ctx, cmd)
		}, "ToggleOwnerUseCase", "ToggleOwner"),
		UpdateStory: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.UpdateStoryCommand) error {
			return updateStoryUseCase.Execute(ctx, cmd)
		}, "UpdateStoryUseCase", "UpdateStory"),
		NewVoting: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.NewVotingCommand) error {
			return newVotingUseCase.Execute(ctx, cmd)
		}, "NewVotingUseCase", "NewVoting"),
		VoteAgain: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.VoteAgainCommand) error {
			return voteAgainUseCase.Execute(ctx, cmd)
		}, "VoteAgainUseCase", "VoteAgain"),
		LeaveRoom: decorator.NewTraceableUseCase(func(ctx context.Context, cmd usecase.LeaveRoomCommand) error {
			return leaveRoomUseCase.Execute(ctx, cmd)
		}, "LeaveRoomUseCase", "LeaveRoom"),
		JoinRoom: decorator.NewTraceableUseCaseWithResult(func(ctx context.Context, cmd usecase.JoinRoomCommand) (*usecase.JoinRoomOutput, error) {
			return joinRoomUseCase.Execute(ctx, cmd)
		}, "JoinRoomUseCase", "JoinRoom"),
	}
}
