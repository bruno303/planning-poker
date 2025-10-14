package main

import (
	"context"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"planning-poker/internal/infra/lock"
	"planning-poker/internal/infra/trace/decorator"
)

type Container struct {
	Hub         *inmemory.InMemoryHub
	LockManager *lock.InMemoryLockManager
	Usecases    usecase.UseCases
}

func NewContainer() *Container {
	hub := inmemory.NewHub()
	lockManager := lock.NewInMemoryLockManager()

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

	return &Container{
		Hub:         hub,
		LockManager: lockManager,
		Usecases: usecase.UseCases{
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
		},
	}
}
