package main

import (
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/infra/boundaries/bus/inmemory"
	"planning-poker/internal/infra/lock"
)

type Container struct {
	Hub         *inmemory.InMemoryHub
	LockManager *lock.InMemoryLockManager
	Usecases    usecase.UseCases
}

func NewContainer() *Container {
	hub := inmemory.NewHub()
	lockManager := lock.NewInMemoryLockManager()

	return &Container{
		Hub:         hub,
		LockManager: lockManager,
		Usecases: usecase.UseCases{
			UpdateName:      usecase.NewUpdateNameUseCase(hub),
			Vote:            usecase.NewVoteUseCase(hub, lockManager),
			Reveal:          usecase.NewRevealUseCase(hub, lockManager),
			Reset:           usecase.NewResetUseCase(hub, lockManager),
			ToggleSpectator: usecase.NewToggleSpectatorUseCase(hub, lockManager),
			ToggleOwner:     usecase.NewToggleOwnerUseCase(hub, lockManager),
			UpdateStory:     usecase.NewUpdateStoryUseCase(hub, lockManager),
			NewVoting:       usecase.NewNewVotingUseCase(hub, lockManager),
			VoteAgain:       usecase.NewVoteAgainUseCase(hub, lockManager),
			LeaveRoom:       usecase.NewLeaveRoomUseCase(hub, lockManager),
			JoinRoom:        usecase.NewJoinRoomUseCase(hub, lockManager),
		},
	}
}
