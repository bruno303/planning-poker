package usecase

type (
	UseCasesFacade struct {
		UpdateName        UseCase[UpdateNameCommand]
		Vote              UseCase[VoteCommand]
		Reveal            UseCase[RevealCommand]
		Reset             UseCase[ResetCommand]
		ToggleSpectator   UseCase[ToggleSpectatorCommand]
		ToggleOwner       UseCase[ToggleOwnerCommand]
		UpdateStory       UseCase[UpdateStoryCommand]
		NewVoting         UseCase[NewVotingCommand]
		VoteAgain         UseCase[VoteAgainCommand]
		LeaveRoom         UseCase[LeaveRoomCommand]
		JoinRoom          UseCaseR[JoinRoomCommand, *JoinRoomOutput]
		CreateClient      UseCaseO[CreateClientOutput]
		CreateRoom        UseCaseO[CreateRoomOutput]
		ToggleBacklogMode UseCase[ToggleBacklogModeCommand]
		AddStory          UseCase[AddStoryCommand]
		RemoveStory       UseCase[RemoveStoryCommand]
		AdvanceStory      UseCase[AdvanceStoryCommand]
		PrevStory         UseCase[PrevStoryCommand]
	}
)
