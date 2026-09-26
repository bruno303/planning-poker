import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import RoomResultsSummary from './roomResultsSummary';
import type { RoomSnapshot } from '@/hooks/room/roomState';

describe('RoomResultsSummary', () => {
  it('shows categorical most-common results without numeric-only fields', () => {
    const snapshot: RoomSnapshot = {
      currentStory: '', reveal: true, result: null, mostAppearingVotes: ['?', '☕'], consensus: null,
      lowestVote: null, highestVote: null, voteRange: null, voteSpread: null, nonNumericVoteCount: 2,
      participants: [], startedAt: null, backlogMode: false, stories: [], currentStoryIndex: 0, roomVersion: 1,
      deck: 't-shirt', deckLabels: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
    };
    render(<RoomResultsSummary snapshot={snapshot} />);
    expect(screen.getByText('Most Common: ?, ☕')).toBeTruthy();
    expect(screen.queryByText(/Average:/)).toBeNull();
    expect(screen.queryByText(/Consensus:/)).toBeNull();
    expect(screen.queryByText(/Non-numeric votes:/)).toBeNull();
  });
});
