package entity

import "time"

// Event is a domain event produced by a Room operation. Events are buffered on
// the aggregate and drained by PullEvents; they are never persisted.
type Event interface {
	// Name returns the stable event name.
	Name() string
	// OccurredAt returns the time at which the event was recorded.
	OccurredAt() time.Time
}

// VoteCast is recorded when a vote is stored on a client. A nil Vote means the
// client retracted its vote.
type VoteCast struct {
	RoomID   string
	ClientID string
	Vote     *Vote
	At       time.Time
}

// Name returns the stable event name.
func (VoteCast) Name() string {
	return "vote-cast"
}

// OccurredAt returns the time at which the event was recorded.
func (e VoteCast) OccurredAt() time.Time {
	return e.At
}

// VotesRevealed is recorded when a hidden round transitions to revealed.
type VotesRevealed struct {
	RoomID  string
	Outcome *RoundOutcome
	At      time.Time
}

// Name returns the stable event name.
func (VotesRevealed) Name() string {
	return "votes-revealed"
}

// OccurredAt returns the time at which the event was recorded.
func (e VotesRevealed) OccurredAt() time.Time {
	return e.At
}

// StorySelected is recorded when the room's current story index moves.
type StorySelected struct {
	RoomID    string
	StoryID   string
	StoryName string
	At        time.Time
}

// Name returns the stable event name.
func (StorySelected) Name() string {
	return "story-selected"
}

// OccurredAt returns the time at which the event was recorded.
func (e StorySelected) OccurredAt() time.Time {
	return e.At
}

// StoryEstimated is recorded when a reveal stamps an estimate onto the current
// backlog story.
type StoryEstimated struct {
	RoomID  string
	StoryID string
	Result  *float32
	At      time.Time
}

// Name returns the stable event name.
func (StoryEstimated) Name() string {
	return "story-estimated"
}

// OccurredAt returns the time at which the event was recorded.
func (e StoryEstimated) OccurredAt() time.Time {
	return e.At
}
