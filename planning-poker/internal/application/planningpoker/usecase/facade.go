package usecase

import (
	"planning-poker/internal/application"
)

type (
	UseCasesFacade struct {
		UpdateName      application.UseCase[UpdateNameCommand]
		Vote            application.UseCase[VoteCommand]
		Reveal          application.UseCase[RevealCommand]
		Reset           application.UseCase[ResetCommand]
		ToggleSpectator application.UseCase[ToggleSpectatorCommand]
		ToggleOwner     application.UseCase[ToggleOwnerCommand]
		UpdateStory     application.UseCase[UpdateStoryCommand]
		NewVoting       application.UseCase[NewVotingCommand]
		VoteAgain       application.UseCase[VoteAgainCommand]
		LeaveRoom       application.UseCase[LeaveRoomCommand]
		JoinRoom        application.UseCaseR[JoinRoomCommand, *JoinRoomOutput]
		CreateRoom      application.UseCaseR[CreateRoomCommand, CreateRoomOutput]
	}
)
