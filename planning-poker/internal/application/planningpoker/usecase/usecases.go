package usecase

import "context"

type (
	UseCase[In any] interface {
		Execute(ctx context.Context, cmd In) error
	}
	UseCaseWithResult[In any, Out any] interface {
		Execute(ctx context.Context, cmd In) (Out, error)
	}

	UseCases struct {
		UpdateName      UseCase[UpdateNameCommand]
		Vote            UseCase[VoteCommand]
		Reveal          UseCase[RevealCommand]
		Reset           UseCase[ResetCommand]
		ToggleSpectator UseCase[ToggleSpectatorCommand]
		ToggleOwner     UseCase[ToggleOwnerCommand]
		UpdateStory     UseCase[UpdateStoryCommand]
		NewVoting       UseCase[NewVotingCommand]
		VoteAgain       UseCase[VoteAgainCommand]
		LeaveRoom       UseCase[LeaveRoomCommand]
		JoinRoom        UseCaseWithResult[JoinRoomCommand, *JoinRoomOutput]
	}
)
