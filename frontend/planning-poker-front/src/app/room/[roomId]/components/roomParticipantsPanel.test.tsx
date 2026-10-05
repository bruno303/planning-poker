import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { RoomSnapshot } from '@/hooks/room/roomState';
import RoomParticipantsPanel from './roomParticipantsPanel';

const tShirtDeck = { id: 't-shirt', name: 'T-shirt', kind: 'categorical', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] };

const snapshot = (overrides: Partial<RoomSnapshot> = {}): RoomSnapshot => ({
  deck: tShirtDeck,
  currentStory: 'Story',
  reveal: true,
  result: null,
  mostAppearingVotes: [],
  consensus: null,
  lowestVote: 'XS',
  highestVote: 'XL',
  voteRange: null,
  voteSpread: 2,
  specialVoteCount: 0,
  participants: [
    { id: 'low', name: 'Ada', vote: 'XS', hasVoted: true, votedAt: null, isSpectator: false, isOwner: true },
    { id: 'high', name: 'Bob', vote: 'XL', hasVoted: true, votedAt: null, isSpectator: false, isOwner: false },
  ],
  startedAt: null,
  backlogMode: false,
  stories: [],
  currentStoryIndex: 0,
  roomVersion: 1,
  ...overrides,
});

const renderPanel = (overrides: Partial<RoomSnapshot> = {}, amIAdmin = false) => {
  const props = {
    onToggleSpectator: vi.fn(),
    onToggleAdmin: vi.fn(),
    onCopied: vi.fn(),
  };

  render(
    <RoomParticipantsPanel
      snapshot={snapshot(overrides)}
      amIAdmin={amIAdmin}
      onToggleSpectator={props.onToggleSpectator}
      onToggleAdmin={props.onToggleAdmin}
      onCopied={props.onCopied}
    />,
  );

  return props;
};

describe('RoomParticipantsPanel', () => {
  afterEach(() => cleanup());

  it('highlights the ordered lowest and highest labels', () => {
    renderPanel();

    expect(screen.getByText('Lowest')).toBeTruthy();
    expect(screen.getByText('Highest')).toBeTruthy();
    expect(screen.getByText('Ada')).toBeTruthy();
    expect(screen.getByText('Bob')).toBeTruthy();
    expect(screen.getByText('XS')).toBeTruthy();
    expect(screen.getByText('XL')).toBeTruthy();
  });

  it('hides votes and extremes before reveal', () => {
    renderPanel({ reveal: false });

    expect(screen.queryByText('Lowest')).toBeNull();
    expect(screen.queryByText('Highest')).toBeNull();
    expect(screen.queryByText('XS')).toBeNull();
    expect(screen.queryAllByText('?').length).toBeGreaterThan(0);
  });

  it('does not highlight special votes as extremes', () => {
    renderPanel({
      participants: [
        { id: 'low', name: 'Ada', vote: 'XS', hasVoted: true, votedAt: null, isSpectator: false, isOwner: true },
        { id: 'special', name: 'Cy', vote: '?', hasVoted: true, votedAt: null, isSpectator: false, isOwner: false },
      ],
    });

    expect(screen.getByText('Lowest')).toBeTruthy();
    expect(screen.queryByText('Highest')).toBeNull();
  });
});
