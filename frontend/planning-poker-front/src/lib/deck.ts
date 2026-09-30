export type DeckId = 'fibonacci' | 'tshirt';
export type DeckKind = 'numeric' | 'categorical';

export type VotingDeck = Readonly<{
  id: DeckId;
  name: string;
  kind: DeckKind;
  cards: readonly string[];
}>;

const deckPresets: Record<DeckId, VotingDeck> = {
  fibonacci: {
    id: 'fibonacci',
    name: 'Fibonacci',
    kind: 'numeric',
    cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'],
  },
  tshirt: {
    id: 'tshirt',
    name: 'T-shirt sizes',
    kind: 'categorical',
    cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
  },
};

export const DECK_OPTIONS: readonly Readonly<{ id: DeckId; name: string }>[] = [
  { id: 'fibonacci', name: deckPresets.fibonacci.name },
  { id: 'tshirt', name: deckPresets.tshirt.name },
];

const categoricalColors = ['#10b981', '#14b8a6', '#0ea5e9', '#3b82f6', '#6366f1', '#a855f7'];
const specialCardColors: Record<string, string> = { '?': '#8b5cf6', '☕': '#f59e0b' };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function matchesCards(value: unknown, expected: readonly string[]): value is string[] {
  return Array.isArray(value) && value.length === expected.length &&
    value.every((card, index) => typeof card === 'string' && card === expected[index]);
}

export function isVotingDeck(value: unknown): value is VotingDeck {
  if (!isRecord(value) || (value.id !== 'fibonacci' && value.id !== 'tshirt')) {
    return false;
  }

  const expected = deckPresets[value.id];
  return value.name === expected.name && value.kind === expected.kind && matchesCards(value.cards, expected.cards);
}

export function normalizeVotingDeck(value: unknown): VotingDeck {
  const deck = value === undefined ? deckPresets.fibonacci : isVotingDeck(value) ? deckPresets[value.id] : null;
  if (!deck) {
    throw new TypeError('Invalid voting deck descriptor');
  }

  return { ...deck, cards: [...deck.cards] };
}

export function normalizeMostCommonVotes(
  mostCommonVotes: readonly string[] | undefined,
  mostAppearingVotes: readonly number[] | null | undefined,
): string[] {
  return mostCommonVotes === undefined ? (mostAppearingVotes ?? []).map(String) : [...mostCommonVotes];
}

export function getVoteCardColor(deck: VotingDeck | null, card: string | null): string {
  if (card === null) {
    return '#9ca3af';
  }

  const specialColor = specialCardColors[card];
  if (specialColor) {
    return specialColor;
  }

  if (deck?.kind === 'numeric') {
    const numericCard = Number(card);
    if (!Number.isFinite(numericCard)) {
      return '#9ca3af';
    }
    if (numericCard <= 2) return '#10b981';
    if (numericCard <= 8) return '#eab308';
    if (numericCard <= 21) return '#f97316';
    return '#ef4444';
  }

  const ordinaryCards = deck?.cards.filter((value) => !(value in specialCardColors)) ?? [];
  const cardIndex = ordinaryCards.indexOf(card);
  return cardIndex < 0 ? '#9ca3af' : categoricalColors[cardIndex % categoricalColors.length];
}
