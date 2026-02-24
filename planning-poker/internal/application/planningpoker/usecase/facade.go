package usecase

type (
	UseCasesFacade struct {
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
		JoinRoom        UseCaseR[JoinRoomCommand, *JoinRoomOutput]
		CreateRoom      UseCaseR[CreateRoomCommand, CreateRoomOutput]
		CreateClient    UseCaseO[CreateClientOutput]
	}
)
