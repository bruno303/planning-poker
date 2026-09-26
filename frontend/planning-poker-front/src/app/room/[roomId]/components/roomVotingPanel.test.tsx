import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import RoomVotingPanel from './roomVotingPanel';
import type { RoomSnapshot } from '@/hooks/room/roomState';

const snapshot: RoomSnapshot = {
  currentStory: 'Story', reveal: false, result: null, mostAppearingVotes: [], consensus: null,
  lowestVote: null, highestVote: null, voteRange: null, voteSpread: null, nonNumericVoteCount: 0,
  participants: [], startedAt: null, backlogMode: false, stories: [], currentStoryIndex: 0, roomVersion: 1,
  deck: 't-shirt', deckLabels: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
};

describe('RoomVotingPanel', () => {
  it('renders the ordered cards from the authoritative room snapshot', () => {
    render(<RoomVotingPanel snapshot={snapshot} clientId="me" userName="Ada" amIAdmin={false} onSelectCard={vi.fn()} onReveal={vi.fn()} onVoteAgain={vi.fn()} onToggleBacklog={vi.fn()} />);
    expect(screen.getAllByRole('button').map((button) => button.textContent)).toEqual(snapshot.deckLabels);
  });
});
