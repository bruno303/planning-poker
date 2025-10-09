package usecase

type UseCases struct {
	UpdateName      UpdateNameUseCase
	Vote            VoteUseCase
	Reveal          RevealUseCase
	Reset           ResetUseCase
	ToggleSpectator ToggleSpectatorUseCase
	ToggleOwner     ToggleOwnerUseCase
	UpdateStory     UpdateStoryUseCase
	NewVoting       NewVotingUseCase
	VoteAgain       VoteAgainUseCase
	LeaveRoom       LeaveRoomUseCase
	JoinRoom        JoinRoomUseCase
}
