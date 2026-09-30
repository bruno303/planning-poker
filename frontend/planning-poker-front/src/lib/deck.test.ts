import { describe, expect, it } from 'vitest';
import { getVoteCardColor, isVotingDeck, normalizeMostCommonVotes, normalizeVotingDeck } from './deck';

const tshirtDeck = {
  id: 'tshirt',
  name: 'T-shirt sizes',
  kind: 'categorical',
  cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
} as const;

describe('voting deck helpers', () => {
  it('defaults missing descriptors to a copied Fibonacci deck', () => {
    const deck = normalizeVotingDeck(undefined);

    expect(deck.id).toBe('fibonacci');
    expect(deck.cards).toEqual(['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕']);
    expect(deck.cards).not.toBe(normalizeVotingDeck(undefined).cards);
  });

  it('accepts known descriptors and rejects malformed or mismatched ones', () => {
    expect(isVotingDeck(tshirtDeck)).toBe(true);
    expect(isVotingDeck({ ...tshirtDeck, cards: ['XS', 'S'] })).toBe(false);
    expect(isVotingDeck({ ...tshirtDeck, kind: 'numeric' })).toBe(false);
    expect(() => normalizeVotingDeck({ ...tshirtDeck, id: 'custom' })).toThrow('Invalid voting deck descriptor');
  });

  it('preserves Fibonacci colors and assigns categorical colors by deck order', () => {
    const fibonacci = normalizeVotingDeck(undefined);
    const tshirt = normalizeVotingDeck(tshirtDeck);

    expect(getVoteCardColor(fibonacci, '0')).toBe('#10b981');
    expect(getVoteCardColor(fibonacci, '5')).toBe('#eab308');
    expect(getVoteCardColor(fibonacci, '?')).toBe('#8b5cf6');
    expect(getVoteCardColor(tshirt, 'XS')).not.toBe(getVoteCardColor(tshirt, 'S'));
    expect(getVoteCardColor(tshirt, '☕')).toBe('#f59e0b');
    expect(getVoteCardColor(null, 'M')).toBe('#9ca3af');
  });

  it('uses canonical modes when present and stringifies legacy numeric modes otherwise', () => {
    expect(normalizeMostCommonVotes(undefined, [5, 8])).toEqual(['5', '8']);
    expect(normalizeMostCommonVotes([], [5, 8])).toEqual([]);
    expect(normalizeMostCommonVotes(['M', 'L'], [5, 8])).toEqual(['M', 'L']);
  });
});
