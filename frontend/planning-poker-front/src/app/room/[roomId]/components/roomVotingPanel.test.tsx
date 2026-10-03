import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { RoomSnapshot } from '@/hooks/room/roomState';
import RoomVotingPanel from './roomVotingPanel';

const fibonacciDeck = { id: 'fibonacci', name: 'Fibonacci', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] };
const tShirtDeck = { id: 't-shirt', name: 'T-shirt', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] };

const snapshot = (overrides: Partial<RoomSnapshot> = {}): RoomSnapshot => ({
  deck: fibonacciDeck,
  currentStory: 'Story',
  reveal: false,
  result: null,
  mostAppearingVotes: [],
  consensus: null,
  lowestVote: null,
  highestVote: null,
  voteRange: null,
  voteSpread: null,
  specialVoteCount: 0,
  participants: [{ id: 'me', name: 'Ada', vote: null, hasVoted: false, votedAt: null, isSpectator: false, isOwner: true }],
  startedAt: null,
  backlogMode: false,
  stories: [],
  currentStoryIndex: 0,
  roomVersion: 1,
  ...overrides,
});

const renderPanel = (overrides: Partial<RoomSnapshot> = {}, amIAdmin = false) => {
  const props = {
    onSelectCard: vi.fn(),
    onReveal: vi.fn(),
    onVoteAgain: vi.fn(),
    onToggleBacklog: vi.fn(),
  };

  render(
    <RoomVotingPanel
      snapshot={snapshot(overrides)}
      clientId="me"
      userName="Ada"
      amIAdmin={amIAdmin}
      onSelectCard={props.onSelectCard}
      onReveal={props.onReveal}
      onVoteAgain={props.onVoteAgain}
      onToggleBacklog={props.onToggleBacklog}
    />,
  );

  return props;
};

describe('RoomVotingPanel', () => {
  afterEach(() => cleanup());

  it('renders the cards supplied by the backend deck descriptor', () => {
    renderPanel();

    for (const card of fibonacciDeck.cards) {
      expect(screen.getByRole('button', { name: card })).toBeTruthy();
    }
  });

  it('renders T-shirt cards instead of Fibonacci cards', () => {
    renderPanel({ deck: tShirtDeck });

    for (const card of tShirtDeck.cards) {
      expect(screen.getByRole('button', { name: card })).toBeTruthy();
    }
    expect(screen.queryByRole('button', { name: '8' })).toBeNull();
  });

  it('selects a card and reflects the acknowledged vote', () => {
    const props = renderPanel({
      participants: [{ id: 'me', name: 'Ada', vote: '8', hasVoted: true, votedAt: null, isSpectator: false, isOwner: true }],
    });

    expect(screen.getByRole('button', { name: '8' }).getAttribute('aria-pressed')).toBe('true');
    fireEvent.click(screen.getByRole('button', { name: '5' }));
    expect(props.onSelectCard).toHaveBeenCalledWith('5');
  });

  it('blocks card selection while votes are revealed', () => {
    const props = renderPanel({ reveal: true });

    fireEvent.click(screen.getByRole('button', { name: '8' }));
    expect(props.onSelectCard).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '8' }).getAttribute('aria-disabled')).toBe('true');
  });

  it('exposes the admin controls', () => {
    const props = renderPanel({}, true);

    fireEvent.click(screen.getByRole('button', { name: /reveal votes/i }));
    fireEvent.click(screen.getByRole('button', { name: /vote again/i }));
    fireEvent.click(screen.getByRole('button', { name: /enable backlog/i }));

    expect(props.onReveal).toHaveBeenCalledOnce();
    expect(props.onVoteAgain).toHaveBeenCalledOnce();
    expect(props.onToggleBacklog).toHaveBeenCalledOnce();
  });
});
