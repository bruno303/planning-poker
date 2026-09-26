package entity

import (
	"math"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestEvaluateVotes(t *testing.T) {
	fibonacci, ok := DeckByID("fibonacci")
	if !ok {
		t.Fatal("fibonacci deck preset not found")
	}
	tshirt, ok := DeckByID("tshirt")
	if !ok {
		t.Fatal("tshirt deck preset not found")
	}

	tests := []struct {
		name  string
		deck  Deck
		cards []string
		want  RoundResult
	}{
		{
			name:  "fibonacci mix with specials",
			deck:  fibonacci,
			cards: []string{"3", "5", "5", "?", "☕", "8"},
			want: RoundResult{
				Average:         lo.ToPtr(5.25),
				MostCommon:      []string{"5"},
				Consensus:       consensusMedium,
				Lowest:          lo.ToPtr(3.0),
				Highest:         lo.ToPtr(8.0),
				Range:           lo.ToPtr(5.0),
				Spread:          lo.ToPtr(2),
				NonNumericCount: 2,
			},
		},
		{
			name:  "decimal votes",
			deck:  fibonacci,
			cards: []string{"1.5", "2.5"},
			want: RoundResult{
				Average:         lo.ToPtr(2.0),
				MostCommon:      []string{"1.5", "2.5"},
				Consensus:       consensusUnavailable,
				Lowest:          lo.ToPtr(1.5),
				Highest:         lo.ToPtr(2.5),
				Range:           lo.ToPtr(1.0),
				Spread:          nil,
				NonNumericCount: 0,
			},
		},
		{
			name:  "tshirt deck only produces most common votes",
			deck:  tshirt,
			cards: []string{"S", "M", "M", "XL"},
			want: RoundResult{
				Average:         nil,
				MostCommon:      []string{"M"},
				Consensus:       consensusUnavailable,
				Lowest:          nil,
				Highest:         nil,
				Range:           nil,
				Spread:          nil,
				NonNumericCount: 4,
			},
		},
		{
			name:  "specials can be the most common vote",
			deck:  fibonacci,
			cards: []string{"?", "?", "☕", "5"},
			want: RoundResult{
				Average:         lo.ToPtr(5.0),
				MostCommon:      []string{"?"},
				Consensus:       consensusHigh,
				Lowest:          lo.ToPtr(5.0),
				Highest:         lo.ToPtr(5.0),
				Range:           lo.ToPtr(0.0),
				Spread:          lo.ToPtr(0),
				NonNumericCount: 3,
			},
		},
		{
			name:  "ties are ordered by deck position",
			deck:  fibonacci,
			cards: []string{"8", "3", "8", "3", "?"},
			want: RoundResult{
				Average:         lo.ToPtr(5.5),
				MostCommon:      []string{"3", "8"},
				Consensus:       consensusMedium,
				Lowest:          lo.ToPtr(3.0),
				Highest:         lo.ToPtr(8.0),
				Range:           lo.ToPtr(5.0),
				Spread:          lo.ToPtr(2),
				NonNumericCount: 1,
			},
		},
		{
			name:  "empty input",
			deck:  fibonacci,
			cards: nil,
			want: RoundResult{
				Average:         nil,
				MostCommon:      []string{},
				Consensus:       consensusUnavailable,
				Lowest:          nil,
				Highest:         nil,
				Range:           nil,
				Spread:          nil,
				NonNumericCount: 0,
			},
		},
		{
			name:  "single vote",
			deck:  fibonacci,
			cards: []string{"5"},
			want: RoundResult{
				Average:         lo.ToPtr(5.0),
				MostCommon:      []string{"5"},
				Consensus:       consensusHigh,
				Lowest:          lo.ToPtr(5.0),
				Highest:         lo.ToPtr(5.0),
				Range:           lo.ToPtr(0.0),
				Spread:          lo.ToPtr(0),
				NonNumericCount: 0,
			},
		},
		{
			name:  "strong majority on adjacent estimates",
			deck:  fibonacci,
			cards: []string{"3", "5", "5"},
			want: RoundResult{
				Average:         lo.ToPtr(13.0 / 3.0),
				MostCommon:      []string{"5"},
				Consensus:       consensusHigh,
				Lowest:          lo.ToPtr(3.0),
				Highest:         lo.ToPtr(5.0),
				Range:           lo.ToPtr(2.0),
				Spread:          lo.ToPtr(1),
				NonNumericCount: 0,
			},
		},
		{
			name:  "large spread",
			deck:  fibonacci,
			cards: []string{"3", "13"},
			want: RoundResult{
				Average:         lo.ToPtr(8.0),
				MostCommon:      []string{"3", "13"},
				Consensus:       consensusLow,
				Lowest:          lo.ToPtr(3.0),
				Highest:         lo.ToPtr(13.0),
				Range:           lo.ToPtr(10.0),
				Spread:          lo.ToPtr(3),
				NonNumericCount: 0,
			},
		},
		{
			name:  "non-deck numeric labels have no spread",
			deck:  fibonacci,
			cards: []string{"3", "4"},
			want: RoundResult{
				Average:         lo.ToPtr(3.5),
				MostCommon:      []string{"3", "4"},
				Consensus:       consensusUnavailable,
				Lowest:          lo.ToPtr(3.0),
				Highest:         lo.ToPtr(4.0),
				Range:           lo.ToPtr(1.0),
				Spread:          nil,
				NonNumericCount: 0,
			},
		},
		{
			name:  "unknown non-deck labels are ordered last",
			deck:  Deck{ID: "custom", Name: "Custom", Cards: []string{"A"}},
			cards: []string{"B", "A"},
			want: RoundResult{
				Average:         nil,
				MostCommon:      []string{"A", "B"},
				Consensus:       consensusUnavailable,
				Lowest:          nil,
				Highest:         nil,
				Range:           nil,
				Spread:          nil,
				NonNumericCount: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateVotes(tt.deck, tt.cards)
			assertRoundResult(t, got, tt.want)
		})
	}
}

func assertRoundResult(t *testing.T, got, want RoundResult) {
	t.Helper()

	assertFloatPointer(t, "Average", got.Average, want.Average)
	assertFloatPointer(t, "Lowest", got.Lowest, want.Lowest)
	assertFloatPointer(t, "Highest", got.Highest, want.Highest)
	assertFloatPointer(t, "Range", got.Range, want.Range)
	if !reflect.DeepEqual(got.MostCommon, want.MostCommon) {
		t.Errorf("MostCommon = %v, want %v", got.MostCommon, want.MostCommon)
	}
	if got.Consensus != want.Consensus {
		t.Errorf("Consensus = %q, want %q", got.Consensus, want.Consensus)
	}
	if !reflect.DeepEqual(got.Spread, want.Spread) {
		t.Errorf("Spread = %v, want %v", got.Spread, want.Spread)
	}
	if got.NonNumericCount != want.NonNumericCount {
		t.Errorf("NonNumericCount = %d, want %d", got.NonNumericCount, want.NonNumericCount)
	}
}

func assertFloatPointer(t *testing.T, name string, got, want *float64) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
		return
	}
	if math.Abs(*got-*want) > 1e-9 {
		t.Errorf("%s = %v, want %v", name, *got, *want)
	}
}
