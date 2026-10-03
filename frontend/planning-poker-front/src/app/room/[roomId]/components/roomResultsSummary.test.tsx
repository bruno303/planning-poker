import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import type { RoomSnapshot } from '@/hooks/room/roomState';
import RoomResultsSummary from './roomResultsSummary';

const fibonacciDeck = { id: 'fibonacci', name: 'Fibonacci', kind: 'numeric', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] };
const tShirtDeck = { id: 't-shirt', name: 'T-shirt', kind: 'categorical', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] };

const snapshot = (overrides: Partial<RoomSnapshot> = {}): RoomSnapshot => ({
  deck: fibonacciDeck,
  currentStory: 'Story',
  reveal: true,
  result: 5.5,
  mostAppearingVotes: ['5'],
  consensus: 'High',
  lowestVote: '3',
  highestVote: '8',
  voteRange: 5,
  voteSpread: 2,
  specialVoteCount: 0,
  participants: [],
  startedAt: null,
  backlogMode: false,
  stories: [],
  currentStoryIndex: 0,
  roomVersion: 1,
  ...overrides,
});

const renderSummary = (overrides: Partial<RoomSnapshot> = {}) => render(<RoomResultsSummary snapshot={snapshot(overrides)} />);

describe('RoomResultsSummary', () => {
  afterEach(() => cleanup());

  it('renders nothing before votes are revealed', () => {
    renderSummary({ reveal: false });

    expect(screen.queryByText('Results Summary')).toBeNull();
  });

  it('renders numeric Fibonacci results', () => {
    renderSummary();

    expect(screen.getByText('Consensus: High')).toBeTruthy();
    expect(screen.getByText('Average: 5.5')).toBeTruthy();
    expect(screen.getByText('Most Common: 5')).toBeTruthy();
    expect(screen.getByText('Votes range from 3 to 8 (spread: 5)')).toBeTruthy();
    expect(screen.getByText('Deck spread: 2 steps')).toBeTruthy();
  });

  it('renders averages for numeric decks without relying on a particular deck ID', () => {
    renderSummary({ deck: { ...fibonacciDeck, id: 'custom-numeric' } });

    expect(screen.getByText('Average: 5.5')).toBeTruthy();
  });

  it('renders ordinal T-shirt results without an average', () => {
    renderSummary({
      deck: tShirtDeck,
      result: null,
      consensus: 'Medium',
      mostAppearingVotes: ['M', 'L'],
      lowestVote: 'S',
      highestVote: 'XL',
      voteRange: null,
      voteSpread: 3,
      specialVoteCount: 2,
    });

    expect(screen.getByText('Consensus: Medium')).toBeTruthy();
    expect(screen.queryByText(/^Average:/)).toBeNull();
    expect(screen.getByText('Most Common: M, L')).toBeTruthy();
    expect(screen.getByText('Votes range from S to XL')).toBeTruthy();
    expect(screen.getByText('Deck spread: 3 steps')).toBeTruthy();
    expect(screen.getByText('Special votes: 2')).toBeTruthy();
  });

  it('omits the special vote count when there are none', () => {
    renderSummary({ specialVoteCount: 0 });

    expect(screen.queryByText(/^Special votes:/)).toBeNull();
  });
});
