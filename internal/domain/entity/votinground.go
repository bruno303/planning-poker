package entity

// VotingRound owns the reveal flag, the current story name and the outcome of
// the round being estimated.
type VotingRound struct {
	currentStory string
	revealed     bool
	outcome      *RoundOutcome
}

// NewVotingRound creates a hidden round without a current story.
func NewVotingRound() *VotingRound {
	return &VotingRound{}
}

// CurrentStoryName returns the name stored for the current story.
func (v *VotingRound) CurrentStoryName() string {
	return v.currentStory
}

// SetCurrentStoryName replaces the name stored for the current story.
func (v *VotingRound) SetCurrentStoryName(name string) {
	v.currentStory = name
}

// Revealed reports whether the round's votes are visible.
func (v *VotingRound) Revealed() bool {
	return v.revealed
}

// Outcome returns the computed outcome, or nil while the round is hidden.
func (v *VotingRound) Outcome() *RoundOutcome {
	if !v.revealed {
		return nil
	}

	return v.outcome
}

// Reveal computes and stores the outcome when the round is hidden, reporting
// whether the round transitioned to revealed. An already revealed round is a
// no-op, so an existing outcome is neither recomputed nor overwritten.
func (v *VotingRound) Reveal(compute func() *RoundOutcome) bool {
	if v.revealed {
		return false
	}

	v.revealed = true
	if compute != nil {
		v.outcome = compute()
	}

	return true
}

// Hide hides the round and always clears the stored outcome.
func (v *VotingRound) Hide() {
	v.revealed = false
	v.outcome = nil
}

// Toggle flips the reveal state and reports whether the round is now revealed.
func (v *VotingRound) Toggle(compute func() *RoundOutcome) bool {
	if v.revealed {
		v.Hide()
		return false
	}

	v.Reveal(compute)
	return true
}
