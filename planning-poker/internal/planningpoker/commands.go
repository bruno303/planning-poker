package planningpoker

type (
	RoomState struct {
		Type         string        `json:"type"`
		CurrentStory string        `json:"currentStory"`
		Reveal       bool          `json:"reveal"`
		Participants []Participant `json:"participants"`
	}
	Participant struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Vote        *string `json:"vote"`
		HasVoted    bool    `json:"hasVoted"`
		IsSpectator bool    `json:"isSpectator"`
		IsOwner     bool    `json:"isOwner"`
	}

	UpdateName struct {
		Type     string `json:"type"`
		Username string `json:"username"`
	}

	UpdateClientID struct {
		Type     string `json:"type"`
		ClientID string `json:"clientId"`
	}
)

func NewRoomStateCommand(room *Room) RoomState {
	return RoomState{
		Type:         "room-state",
		CurrentStory: room.CurrentStory,
		Reveal:       room.Reveal,
		Participants: MapToParticipants(room.Clients, room.OwnerClient),
	}
}

func NewUpdateNameCommand(username string) UpdateName {
	return UpdateName{
		Type:     "update-name",
		Username: username,
	}
}

func NewUpdateClientIDCommand(clientID string) UpdateClientID {
	return UpdateClientID{
		Type:     "update-client-id",
		ClientID: clientID,
	}
}
