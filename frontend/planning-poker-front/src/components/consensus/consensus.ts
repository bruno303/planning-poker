export type ExtremeVote = 'lowest' | 'highest';

export const SPECIAL_CARD_COLORS: Record<string, string> = {
  '?': '#8b5cf6',
  '☕': '#f59e0b',
};

const ESTIMATE_COLOR_STOPS = [
  { maxRatio: 0.3, color: '#10b981' },
  { maxRatio: 0.55, color: '#eab308' },
  { maxRatio: 0.8, color: '#f97316' },
];

const FALLBACK_CARD_COLOR = '#ef4444';
const UNKNOWN_CARD_COLOR = '#6b7280';

export function getExtremeVotes(
  vote: string | null,
  lowestVote: string | null,
  highestVote: string | null,
): ExtremeVote[] {
  if (vote === null || lowestVote === null || highestVote === null) {
    return [];
  }

  const trimmedVote = vote.trim();
  const trimmedLowest = lowestVote.trim();
  const trimmedHighest = highestVote.trim();
  if (!trimmedVote || trimmedLowest === trimmedHighest) {
    return [];
  }

  const extremes: ExtremeVote[] = [];
  if (trimmedVote === trimmedLowest) {
    extremes.push('lowest');
  }
  if (trimmedVote === trimmedHighest) {
    extremes.push('highest');
  }

  return extremes;
}

// Derives a presentation color from the card's ordered position in the deck.
// Special cards keep an explicit color and unknown cards fall back to gray.
export function getCardColor(card: string | null, cards: string[]): string {
  if (card === null) {
    return UNKNOWN_CARD_COLOR;
  }
  if (card in SPECIAL_CARD_COLORS) {
    return SPECIAL_CARD_COLORS[card];
  }

  const estimates = cards.filter((estimate) => !(estimate in SPECIAL_CARD_COLORS));
  const index = estimates.indexOf(card);
  if (index < 0 || estimates.length <= 1) {
    return UNKNOWN_CARD_COLOR;
  }

  const ratio = index / (estimates.length - 1);
  for (const stop of ESTIMATE_COLOR_STOPS) {
    if (ratio < stop.maxRatio) {
      return stop.color;
    }
  }

  return FALLBACK_CARD_COLOR;
}
