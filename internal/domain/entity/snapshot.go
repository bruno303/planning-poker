package entity

import "time"

type (
	// RoomSnapshot is an immutable-value view of a room used to clone and
	// rebuild aggregates without exposing their internals.
	RoomSnapshot struct {
		ID                 string
		StartedAt          time.Time
		DeckType           DeckType
		Version            uint64
		Clients            []ClientSnapshot
		CurrentStory       string
		Revealed           bool
		Result             *float32
		MostAppearingVotes []string
		Consensus          string
		LowestVote         *string
		HighestVote        *string
		VoteRange          *int
		VoteSpread         *int
		SpecialVoteCount   int
		BacklogMode        bool
		Stories            []Story
		CurrentStoryIndex  int
	}
	ClientSnapshot struct {
		ID          string
		Name        string
		Vote        *Vote
		HasVoted    bool
		VotedAt     *time.Time
		IsSpectator bool
		IsOwner     bool
	}
)

// Snapshot returns a deep copy of the room state.
func (r *Room) Snapshot() RoomSnapshot {
	clients := make([]ClientSnapshot, 0, r.presence.Count())
	for _, client := range r.presence.Values() {
		clients = append(clients, ClientSnapshot{
			ID:          client.ID,
			Name:        client.Name,
			Vote:        cloneVote(client.CurrentVote),
			HasVoted:    client.HasVoted,
			VotedAt:     OptionalUTCTimePtr(client.VotedAt),
			IsSpectator: client.IsSpectator,
			IsOwner:     client.IsOwner,
		})
	}

	return RoomSnapshot{
		ID:                 r.id,
		StartedAt:          r.startedAt,
		DeckType:           r.DeckType,
		Version:            r.RoomVersion,
		Clients:            clients,
		CurrentStory:       r.CurrentStory,
		Revealed:           r.Reveal,
		Result:             cloneFloat32(r.Result),
		MostAppearingVotes: append([]string(nil), r.MostAppearingVotes...),
		Consensus:          r.Consensus,
		LowestVote:         cloneString(r.LowestVote),
		HighestVote:        cloneString(r.HighestVote),
		VoteRange:          cloneInt(r.VoteRange),
		VoteSpread:         cloneInt(r.VoteSpread),
		SpecialVoteCount:   r.SpecialVoteCount,
		BacklogMode:        r.BacklogMode,
		Stories:            cloneStories(r.Stories),
		CurrentStoryIndex:  r.CurrentStoryIndex,
	}
}

// NewRoomFromSnapshot rebuilds a room from a snapshot. Clients are re-added
// through the presence and their persisted flags then overwrite the defaults.
func NewRoomFromSnapshot(s RoomSnapshot, clients ClientCollection) *Room {
	room := &Room{
		id:                 s.ID,
		startedAt:          s.StartedAt.UTC(),
		presence:           NewPresence(s.ID, clients),
		DeckType:           s.DeckType,
		CurrentStory:       s.CurrentStory,
		Reveal:             s.Revealed,
		Result:             cloneFloat32(s.Result),
		MostAppearingVotes: append([]string(nil), s.MostAppearingVotes...),
		Consensus:          s.Consensus,
		LowestVote:         cloneString(s.LowestVote),
		HighestVote:        cloneString(s.HighestVote),
		VoteRange:          cloneInt(s.VoteRange),
		VoteSpread:         cloneInt(s.VoteSpread),
		SpecialVoteCount:   s.SpecialVoteCount,
		BacklogMode:        s.BacklogMode,
		Stories:            cloneStories(s.Stories),
		CurrentStoryIndex:  s.CurrentStoryIndex,
		RoomVersion:        s.Version,
	}

	for _, snapshot := range s.Clients {
		client := room.presence.Add(snapshot.ID)
		client.Name = snapshot.Name
		client.CurrentVote = cloneVote(snapshot.Vote)
		client.HasVoted = snapshot.HasVoted
		client.VotedAt = OptionalUTCTimePtr(snapshot.VotedAt)
		client.IsSpectator = snapshot.IsSpectator
		client.IsOwner = snapshot.IsOwner
	}

	return room
}

func cloneVote(vote *Vote) *Vote {
	if vote == nil {
		return nil
	}

	cloned := *vote
	return &cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneFloat32(value *float32) *float32 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneStories(stories []Story) []Story {
	if stories == nil {
		return nil
	}

	cloned := make([]Story, len(stories))
	for i, story := range stories {
		cloned[i] = story
		cloned[i].Result = cloneFloat32(story.Result)
		cloned[i].MostAppearingVotes = append([]string(nil), story.MostAppearingVotes...)
	}

	return cloned
}
