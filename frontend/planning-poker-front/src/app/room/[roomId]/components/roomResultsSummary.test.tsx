import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import type { RoomSnapshot } from '@/hooks/room/roomState';
import RoomResultsSummary from './roomResultsSummary';

const snapshot = (overrides: Partial<RoomSnapshot> = {}): RoomSnapshot => ({
  currentStory: 'Story',
  reveal: true,
  result: null,
  mostAppearingVotes: [],
  consensus: null,
  lowestVote: null,
  highestVote: null,
  voteRange: null,
  voteSpread: null,
  nonNumericVoteCount: 0,
  participants: [],
  startedAt: null,
  backlogMode: false,
  stories: [],
  currentStoryIndex: 0,
  roomVersion: 1,
  deck: [],
  deckPreset: null,
  ...overrides,
});

describe('RoomResultsSummary', () => {
  afterEach(() => cleanup());

  it('shows only the most common votes for a non-numeric deck', () => {
    render(<RoomResultsSummary snapshot={snapshot({
      deck: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
      deckPreset: 'tshirt',
      mostAppearingVotes: ['M'],
      consensus: 'Unavailable',
      nonNumericVoteCount: 2,
    })} />);

    expect(screen.getByText('Most Common: M')).toBeTruthy();
    expect(screen.queryByText(/Unavailable/)).toBeNull();
    expect(screen.queryByText(/Average:/)).toBeNull();
    expect(screen.queryByText(/Consensus:/)).toBeNull();
    expect(screen.queryByText(/Votes range/)).toBeNull();
    expect(screen.queryByText(/Deck spread/)).toBeNull();
    expect(screen.queryByText(/Non-numeric votes:/)).toBeNull();
  });

  it('shows every applicable row for a numeric reveal', () => {
    render(<RoomResultsSummary snapshot={snapshot({
      mostAppearingVotes: ['5', '8'],
      consensus: 'High',
      result: 5.5,
      lowestVote: 3,
      highestVote: 8,
      voteRange: 5,
      voteSpread: 2,
      nonNumericVoteCount: 1,
    })} />);

    expect(screen.getByText('Consensus: High')).toBeTruthy();
    expect(screen.getByText('Average: 5.5')).toBeTruthy();
    expect(screen.getByText('Most Common: 5, 8')).toBeTruthy();
    expect(screen.getByText('Votes range from 3 to 8 (spread: 5)')).toBeTruthy();
    expect(screen.getByText('Deck spread: 2 steps')).toBeTruthy();
    expect(screen.getByText('Non-numeric votes: 1')).toBeTruthy();
  });

  it('hides the panel when no metric row would render', () => {
    render(<RoomResultsSummary snapshot={snapshot()} />);

    expect(screen.queryByText('Results Summary')).toBeNull();
  });

  it('hides the panel before the reveal', () => {
    render(<RoomResultsSummary snapshot={snapshot({ reveal: false, result: 5 })} />);

    expect(screen.queryByText('Results Summary')).toBeNull();
  });
});
