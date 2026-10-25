package entity

import (
	"slices"
	"time"
)

type (
	// RoomSnapshot is an immutable-value view of a room used to clone and
	// rebuild aggregates without exposing their internals.
	RoomSnapshot struct {
		ID                string
		StartedAt         time.Time
		DeckType          DeckType
		Version           RoomVersion
		Clients           []ClientSnapshot
		CurrentStory      string
		Revealed          bool
		Outcome           *RoundOutcome
		BacklogMode       bool
		Stories           []Story
		CurrentStoryIndex int
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
		ID:                r.id,
		StartedAt:         r.startedAt,
		DeckType:          r.deckType,
		Version:           r.version,
		Clients:           clients,
		CurrentStory:      r.round.CurrentStoryName(),
		Revealed:          r.round.Revealed(),
		Outcome:           r.round.Outcome().clone(),
		BacklogMode:       r.backlog.Mode(),
		Stories:           cloneStories(r.backlog.Stories()),
		CurrentStoryIndex: r.backlog.CurrentIndex(),
	}
}

// NewRoomFromSnapshot rebuilds a room from a snapshot. Clients are re-added
// through the presence and their persisted flags then overwrite the defaults.
func NewRoomFromSnapshot(s RoomSnapshot, clients ClientCollection) *Room {
	round := NewVotingRound()
	round.SetCurrentStoryName(s.CurrentStory)
	// Restoring through Reveal keeps a hidden snapshot from carrying an
	// outcome, mirroring Hide.
	if s.Revealed {
		round.Reveal(func() *RoundOutcome { return s.Outcome.clone() })
	}

	backlog := NewBacklog(s.ID)
	backlog.restore(s.BacklogMode, cloneStories(s.Stories), s.CurrentStoryIndex)

	room := &Room{
		id:        s.ID,
		startedAt: s.StartedAt.UTC(),
		presence:  NewPresence(s.ID, clients),
		round:     round,
		backlog:   backlog,
		deckType:  s.DeckType,
		version:   s.Version,
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
		cloned[i].MostAppearingVotes = slices.Clone(story.MostAppearingVotes)
	}

	return cloned
}
