package entity

import "time"

// Event is a domain event recorded on a room aggregate.
type Event interface {
	Name() string
}

const (
	eventNameVoteCast       = "vote-cast"
	eventNameVotesRevealed  = "votes-revealed"
	eventNameStorySelected  = "story-selected"
	eventNameStoryEstimated = "story-estimated"
)

// VoteCast records a vote applied by a client.
type VoteCast struct {
	RoomID     string
	ClientID   ClientID
	Vote       Vote
	OccurredAt time.Time
}

// Name returns the event name.
func (VoteCast) Name() string {
	return eventNameVoteCast
}

// VotesRevealed records the hidden to revealed transition of a voting round.
type VotesRevealed struct {
	RoomID     string
	Result     *float32
	Consensus  string
	OccurredAt time.Time
}

// Name returns the event name.
func (VotesRevealed) Name() string {
	return eventNameVotesRevealed
}

// StorySelected records a backlog story becoming the current story.
type StorySelected struct {
	RoomID     string
	StoryID    StoryID
	OccurredAt time.Time
}

// Name returns the event name.
func (StorySelected) Name() string {
	return eventNameStorySelected
}

// StoryEstimated records the current story being stamped as estimated.
type StoryEstimated struct {
	RoomID     string
	StoryID    StoryID
	Result     *float32
	OccurredAt time.Time
}

// Name returns the event name.
func (StoryEstimated) Name() string {
	return eventNameStoryEstimated
}

// cloneResult copies a result value so events never alias room state.
func cloneResult(result *float32) *float32 {
	if result == nil {
		return nil
	}
	value := *result
	return &value
}
