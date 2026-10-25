package entity

import (
	"slices"
	"strconv"

	"github.com/samber/lo"
)

// RoundOutcome is the immutable-value summary computed when a voting round is
// revealed.
type RoundOutcome struct {
	Result             *float32
	MostAppearingVotes []string
	Consensus          string
	LowestVote         *string
	HighestVote        *string
	VoteRange          *int
	VoteSpread         *int
	SpecialVoteCount   int
}

// Revealed reports whether the current round's votes are visible.
func (r *Room) Revealed() bool {
	return r.round.Revealed()
}

// Outcome returns the current round's outcome, or nil while the round is
// hidden.
func (r *Room) Outcome() *RoundOutcome {
	return r.round.Outcome()
}

// Result returns the mean of the numeric votes, or nil when unavailable.
func (r *Room) Result() *float32 {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.Result
	}

	return nil
}

// MostAppearingVotes returns the most frequent votes ordered by deck position.
func (r *Room) MostAppearingVotes() []string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.MostAppearingVotes
	}

	return nil
}

// Consensus returns the consensus level of the current round.
func (r *Room) Consensus() string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.Consensus
	}

	return ""
}

// LowestVote returns the lowest vote by deck position.
func (r *Room) LowestVote() *string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.LowestVote
	}

	return nil
}

// HighestVote returns the highest vote by deck position.
func (r *Room) HighestVote() *string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.HighestVote
	}

	return nil
}

// VoteRange returns the numeric distance between the extreme votes.
func (r *Room) VoteRange() *int {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.VoteRange
	}

	return nil
}

// VoteSpread returns the deck-position distance between the extreme votes.
func (r *Room) VoteSpread() *int {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.VoteSpread
	}

	return nil
}

// SpecialVoteCount returns how many special cards were cast.
func (r *Room) SpecialVoteCount() int {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.SpecialVoteCount
	}

	return 0
}

// clone returns a deep copy of the outcome. A nil outcome stays nil.
func (o *RoundOutcome) clone() *RoundOutcome {
	if o == nil {
		return nil
	}

	return &RoundOutcome{
		Result:             cloneFloat32(o.Result),
		MostAppearingVotes: slices.Clone(o.MostAppearingVotes),
		Consensus:          o.Consensus,
		LowestVote:         cloneString(o.LowestVote),
		HighestVote:        cloneString(o.HighestVote),
		VoteRange:          cloneInt(o.VoteRange),
		VoteSpread:         cloneInt(o.VoteSpread),
		SpecialVoteCount:   o.SpecialVoteCount,
	}
}

// computeOutcome summarises votes in client iteration order. It always returns
// a non-nil outcome, mirroring a revealed round without usable votes.
func computeOutcome(deck Deck, votes []Vote) *RoundOutcome {
	metrics := collectVotes(deck, votes)
	summary := calculateConsensus(deck, metrics.ordered)

	outcome := &RoundOutcome{
		MostAppearingVotes: mostAppearingVotes(metrics.counts, getMostVoteCount(metrics.counts), deck),
		Consensus:          summary.Consensus,
		LowestVote:         summary.LowestVote,
		HighestVote:        summary.HighestVote,
		VoteRange:          summary.VoteRange,
		VoteSpread:         summary.VoteSpread,
		SpecialVoteCount:   metrics.specialCount,
	}
	if metrics.count > 0 {
		outcome.Result = lo.ToPtr(metrics.sum / metrics.count)
	}

	return outcome
}

type voteMetrics struct {
	sum          float32
	count        float32
	counts       map[string]int
	ordered      []string
	specialCount int
}

func collectVotes(deck Deck, votes []Vote) voteMetrics {
	metrics := voteMetrics{counts: make(map[string]int)}

	for _, vote := range votes {
		label := vote.Label()
		if label == "" {
			continue
		}

		if _, ok := deck.Position(label); !ok {
			metrics.specialCount++
			continue
		}

		if value, err := strconv.Atoi(label); err == nil {
			metrics.sum += float32(value)
			metrics.count++
		}
		metrics.counts[label]++
		metrics.ordered = append(metrics.ordered, label)
	}

	return metrics
}

func mostAppearingVotes(votes map[string]int, mostVoteCount int, deck Deck) []string {
	mostVotes := make([]string, 0)
	for vote, count := range votes {
		if count == mostVoteCount {
			mostVotes = append(mostVotes, vote)
		}
	}
	slices.SortFunc(mostVotes, func(a, b string) int {
		aPosition, _ := deck.Position(a)
		bPosition, _ := deck.Position(b)
		return aPosition - bPosition
	})

	return mostVotes
}

func getMostVoteCount(voteMap map[string]int) int {
	var mostVoteCount int
	for _, count := range voteMap {
		if count > mostVoteCount {
			mostVoteCount = count
		}
	}

	return mostVoteCount
}

const (
	consensusHigh        = "High"
	consensusMedium      = "Medium"
	consensusLow         = "Low"
	consensusUnavailable = "Unavailable"
)

type voteSummary struct {
	Consensus   string
	LowestVote  *string
	HighestVote *string
	VoteRange   *int
	VoteSpread  *int
}

// calculateConsensus evaluates ordered vote labels using deck positions.
// VoteRange is a numeric distance only for numeric decks; VoteSpread always
// measures the distance between ordered deck positions.
func calculateConsensus(deck Deck, votes []string) voteSummary {
	if len(votes) == 0 {
		return voteSummary{Consensus: consensusUnavailable}
	}

	voteCounts := make(map[string]int, len(votes))
	for _, vote := range votes {
		voteCounts[vote]++
	}

	minPosition, maxPosition := -1, -1
	var minLabel, maxLabel string
	for _, vote := range votes {
		position, ok := deck.Position(vote)
		if !ok {
			continue
		}
		if minPosition == -1 || position < minPosition {
			minPosition, minLabel = position, vote
		}
		if position > maxPosition {
			maxPosition, maxLabel = position, vote
		}
	}

	summary := voteSummary{Consensus: consensusUnavailable}
	if minPosition == -1 {
		return summary
	}

	spread := maxPosition - minPosition
	summary.VoteSpread = lo.ToPtr(spread)
	summary.LowestVote = lo.ToPtr(minLabel)
	summary.HighestVote = lo.ToPtr(maxLabel)
	summary.Consensus = consensusLow

	mostVoteCount := getMostVoteCount(voteCounts)
	strongMajority := mostVoteCount >= (2*len(votes)+2)/3
	switch {
	case minLabel == maxLabel:
		summary.Consensus = consensusHigh
	case strongMajority && spread <= 1:
		summary.Consensus = consensusHigh
	case spread <= 2:
		summary.Consensus = consensusMedium
	}

	if minValue, minErr := strconv.Atoi(minLabel); minErr == nil {
		if maxValue, maxErr := strconv.Atoi(maxLabel); maxErr == nil {
			summary.VoteRange = lo.ToPtr(maxValue - minValue)
		}
	}

	return summary
}
