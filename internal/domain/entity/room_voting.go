package entity

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/samber/lo"
)

func (r *Room) Vote(ctx context.Context, clientID ClientID, vote *Vote) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}

	if vote != nil && !vote.IsEmpty() {
		deck := r.Deck()
		if !deck.HasCard(vote.String()) {
			return fmt.Errorf("vote %q is not part of deck %s", vote.String(), deck.ID)
		}
	}

	applied := r.applyVote(client, vote)
	if applied && vote != nil && !vote.IsEmpty() {
		r.recordEvent(VoteCast{
			RoomID:     r.ID,
			ClientID:   clientID,
			Vote:       *vote,
			OccurredAt: time.Now().UTC(),
		})
	}

	r.checkReveal()

	return nil
}

func (r *Room) applyVote(client *Client, vote *Vote) bool {
	if r.Reveal {
		return false
	}

	client.applyVote(vote)
	return true
}

func (r *Room) ToggleReveal(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(ClientID(clientID))
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle reveal")
	}

	r.reveal(!r.Reveal)
	return nil
}

func (r *Room) ResetVoting(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(ClientID(clientID))
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can reset voting")
	}

	r.reveal(false)

	r.Clients.ForEach(func(c *Client) {
		r.applyVote(c, nil)
	})
	return nil
}

func (r *Room) reveal(reveal bool) {
	if reveal && r.Reveal {
		return
	}

	r.Reveal = reveal

	if !reveal {
		r.clearConsensus()
		return
	}

	deck := r.Deck()
	metrics := r.collectVotes(deck)
	r.MostAppearingVotes = mostAppearingVotes(metrics.counts, getMostVoteCount(metrics.counts), deck)

	if metrics.count > 0 {
		r.Result = lo.ToPtr(metrics.sum / metrics.count)
	} else {
		r.Result = nil
	}

	summary := calculateConsensus(deck, metrics.ordered)
	r.Consensus = summary.Consensus
	r.LowestVote = summary.LowestVote
	r.HighestVote = summary.HighestVote
	r.VoteRange = summary.VoteRange
	r.VoteSpread = summary.VoteSpread
	r.SpecialVoteCount = metrics.specialCount

	var estimatedStoryID StoryID
	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Result = r.Result
		r.Stories[r.CurrentStoryIndex].MostAppearingVotes = r.MostAppearingVotes
		r.Stories[r.CurrentStoryIndex].Voted = true
		estimatedStoryID = r.Stories[r.CurrentStoryIndex].ID
	}

	r.recordEvent(VotesRevealed{
		RoomID:     r.ID,
		Result:     cloneResult(r.Result),
		Consensus:  r.Consensus,
		OccurredAt: time.Now().UTC(),
	})

	if estimatedStoryID != "" {
		r.recordEvent(StoryEstimated{
			RoomID:     r.ID,
			StoryID:    estimatedStoryID,
			Result:     cloneResult(r.Result),
			OccurredAt: time.Now().UTC(),
		})
	}
}

func (r *Room) collectVotes(deck Deck) voteMetrics {
	metrics := voteMetrics{counts: make(map[string]int)}

	for _, client := range r.Clients.Values() {
		if client.IsSpectator || client.CurrentVote == nil {
			continue
		}

		vote := client.CurrentVote.String()
		if vote == "" {
			continue
		}

		if _, ok := deck.Position(vote); !ok {
			metrics.specialCount++
			continue
		}

		if value, err := strconv.Atoi(vote); err == nil {
			metrics.sum += float32(value)
			metrics.count++
		}
		metrics.counts[vote]++
		metrics.ordered = append(metrics.ordered, vote)
	}

	return metrics
}

type voteMetrics struct {
	sum          float32
	count        float32
	counts       map[string]int
	ordered      []string
	specialCount int
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

func (r *Room) clearConsensus() {
	r.Result = nil
	r.MostAppearingVotes = nil
	r.Consensus = ""
	r.LowestVote = nil
	r.HighestVote = nil
	r.VoteRange = nil
	r.VoteSpread = nil
	r.SpecialVoteCount = 0
}
