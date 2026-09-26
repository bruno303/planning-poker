package entity

import (
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

const (
	consensusHigh        = "High"
	consensusMedium      = "Medium"
	consensusLow         = "Low"
	consensusUnavailable = "Unavailable"
)

// RoundResult is the interpretation of the cards voted in a revealed round.
type RoundResult struct {
	Average         *float64
	MostCommon      []string
	Consensus       string
	Lowest          *float64
	Highest         *float64
	Range           *float64
	Spread          *int
	NonNumericCount int
}

// EvaluateVotes interprets cards against the room deck: it counts every card,
// parses numeric votes, and derives consensus. It is a pure function.
func EvaluateVotes(deck Deck, cards []string) RoundResult {
	var (
		counts           = make(map[string]int, len(cards))
		numericCounts    = make(map[float64]int, len(cards))
		mostNumericVotes int
		nonNumericCount  int
		sum              float64
		lowest           float64
		highest          float64
		lowestLabel      string
		highestLabel     string
		numericVotes     int
	)

	for _, card := range cards {
		counts[card]++

		value, err := strconv.ParseFloat(strings.TrimSpace(card), 64)
		if err != nil {
			nonNumericCount++
			continue
		}

		numericVotes++
		numericCounts[value]++
		if numericCounts[value] > mostNumericVotes {
			mostNumericVotes = numericCounts[value]
		}
		sum += value

		if numericVotes == 1 {
			lowest, highest = value, value
			lowestLabel, highestLabel = card, card
			continue
		}
		if value < lowest {
			lowest = value
			lowestLabel = card
		}
		if value > highest {
			highest = value
			highestLabel = card
		}
	}

	result := RoundResult{
		MostCommon:      mostCommonCards(deck, counts),
		Consensus:       consensusUnavailable,
		NonNumericCount: nonNumericCount,
	}
	if numericVotes == 0 {
		return result
	}

	result.Average = lo.ToPtr(sum / float64(numericVotes))
	result.Lowest = lo.ToPtr(lowest)
	result.Highest = lo.ToPtr(highest)
	result.Range = lo.ToPtr(highest - lowest)

	lowestPosition, lowestKnown := deck.Position(strings.TrimSpace(lowestLabel))
	highestPosition, highestKnown := deck.Position(strings.TrimSpace(highestLabel))
	if lowestKnown && highestKnown {
		result.Spread = lo.ToPtr(highestPosition - lowestPosition)
	}

	switch {
	case lowest == highest:
		result.Consensus = consensusHigh
	case result.Spread != nil && mostNumericVotes >= (2*numericVotes+2)/3 && *result.Spread <= 1:
		result.Consensus = consensusHigh
	case result.Spread != nil && *result.Spread <= 2:
		result.Consensus = consensusMedium
	case result.Spread != nil:
		result.Consensus = consensusLow
	}

	return result
}

func mostCommonCards(deck Deck, counts map[string]int) []string {
	mostCount := 0
	for _, count := range counts {
		if count > mostCount {
			mostCount = count
		}
	}

	mostCommon := make([]string, 0, len(counts))
	for card, count := range counts {
		if count == mostCount {
			mostCommon = append(mostCommon, card)
		}
	}

	slices.SortFunc(mostCommon, func(a, b string) int {
		aPosition, aKnown := deck.Position(a)
		bPosition, bKnown := deck.Position(b)
		if aKnown && bKnown {
			return aPosition - bPosition
		}
		if aKnown != bKnown {
			if aKnown {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})

	return mostCommon
}
