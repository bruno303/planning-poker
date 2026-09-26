import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { RoomSnapshot } from '@/hooks/room/roomState';
import RoomVotingPanel from './roomVotingPanel';

const snapshot = (overrides: Partial<RoomSnapshot> = {}): RoomSnapshot => ({
  currentStory: 'Story',
  reveal: false,
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
  deck: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
  deckPreset: 'tshirt',
  ...overrides,
});

const renderPanel = (overrides: Partial<RoomSnapshot> = {}, onSelectCard = vi.fn()) => {
  render(<RoomVotingPanel
    snapshot={snapshot(overrides)}
    clientId="me"
    userName="Ada"
    amIAdmin={false}
    onSelectCard={onSelectCard}
    onReveal={vi.fn()}
    onVoteAgain={vi.fn()}
    onToggleBacklog={vi.fn()}
  />);
  return onSelectCard;
};

describe('RoomVotingPanel', () => {
  afterEach(() => cleanup());

  it('renders the cards from the room snapshot deck', () => {
    renderPanel();

    for (const card of ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕']) {
      expect(screen.getByRole('button', { name: card })).toBeTruthy();
    }
    expect(screen.queryByRole('button', { name: '13' })).toBeNull();
  });

  it('falls back to the Fibonacci deck when the snapshot deck is empty', () => {
    renderPanel({ deck: [], deckPreset: null });

    expect(screen.getByRole('button', { name: '13' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'M' })).toBeNull();
  });

  it('sends the selected card', () => {
    const onSelectCard = renderPanel({ deck: ['5', '8'] });

    fireEvent.click(screen.getByRole('button', { name: '8' }));

    expect(onSelectCard).toHaveBeenCalledWith('8');
  });

  it('disables card selection after the votes are revealed', () => {
    const onSelectCard = renderPanel({ reveal: true, deck: ['5', '8'] });
    const card = screen.getByRole('button', { name: '8' });

    expect(card.getAttribute('aria-disabled')).toBe('true');
    expect(card.style.backgroundColor).toBe('rgb(156, 163, 175)');
    fireEvent.click(card);
    expect(onSelectCard).not.toHaveBeenCalled();
  });
});
