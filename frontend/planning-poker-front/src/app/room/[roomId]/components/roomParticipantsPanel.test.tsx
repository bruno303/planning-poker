import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import RoomParticipantsPanel from './roomParticipantsPanel';
import type { RoomSnapshot } from '@/hooks/room/roomState';

describe('RoomParticipantsPanel', () => {
  it('safely displays categorical revealed participant votes', () => {
    const snapshot: RoomSnapshot = {
      currentStory: '', reveal: true, result: null, mostAppearingVotes: ['☕'], consensus: null,
      lowestVote: null, highestVote: null, voteRange: null, voteSpread: null, nonNumericVoteCount: 0,
      participants: [{ id: 'p1', name: 'Ada', vote: '☕', hasVoted: true, votedAt: null, isSpectator: false, isOwner: false }],
      startedAt: null, backlogMode: false, stories: [], currentStoryIndex: 0, roomVersion: 1,
      deck: 't-shirt', deckLabels: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
    };
    render(<RoomParticipantsPanel snapshot={snapshot} amIAdmin={false} onToggleSpectator={vi.fn()} onToggleAdmin={vi.fn()} onCopied={vi.fn()} />);
    const voteCard = screen.getByText('☕');
    expect(voteCard).toBeTruthy();
    expect(voteCard).toHaveProperty('style.backgroundColor', 'rgb(245, 158, 11)');
  });
});
