import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import type { RoomSnapshot } from '@/hooks/room/roomState';
import RoomResultsSummary from './roomResultsSummary';

const snapshot = (overrides: Partial<RoomSnapshot> = {}): RoomSnapshot => ({
  deck: { id: 'fibonacci', name: 'Fibonacci', kind: 'numeric', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] },
  currentStory: 'Story',
  reveal: true,
  result: 5.5,
  mostAppearingVotes: [5, 8],
  mostCommonVotes: ['5', '8'],
  consensus: 'Medium',
  lowestVote: 3,
  highestVote: 8,
  voteRange: 5,
  voteSpread: 2,
  nonNumericVoteCount: 1,
  participants: [],
  startedAt: null,
  backlogMode: false,
  stories: [],
  currentStoryIndex: 0,
  roomVersion: 4,
  ...overrides,
});

describe('RoomResultsSummary', () => {
  afterEach(() => cleanup());

  it('keeps numeric summary statistics for Fibonacci rooms', () => {
    render(<RoomResultsSummary snapshot={snapshot()} />);

    expect(screen.getByText('Average: 5.5')).toBeTruthy();
    expect(screen.getByText('Most Common: 5, 8')).toBeTruthy();
    expect(screen.getByText('Deck spread: 2 steps')).toBeTruthy();
    expect(screen.getByText('Non-numeric votes: 1')).toBeTruthy();
  });

  it('shows tied T-shirt modes without numeric statistics', () => {
    render(<RoomResultsSummary snapshot={snapshot({
      deck: { id: 'tshirt', name: 'T-shirt sizes', kind: 'categorical', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] },
      result: null,
      mostCommonVotes: ['M', 'L'],
      consensus: 'Unavailable',
      lowestVote: null,
      highestVote: null,
      voteRange: null,
      voteSpread: null,
    })} />);

    expect(screen.getByText('Consensus: Unavailable')).toBeTruthy();
    expect(screen.getByText('Most Common: M, L')).toBeTruthy();
    expect(screen.queryByText(/Average:/)).toBeNull();
    expect(screen.queryByText(/Votes range/)).toBeNull();
    expect(screen.queryByText(/Deck spread/)).toBeNull();
    expect(screen.queryByText(/Non-numeric votes/)).toBeNull();
  });
});
