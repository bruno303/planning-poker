import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { RoomSnapshot } from '@/hooks/room/roomState';
import { normalizeVotingDeck } from '@/lib/deck';
import RoomVotingPanel from './roomVotingPanel';

vi.mock('@/components/avatar/avatar', () => ({ default: () => <span aria-hidden="true" /> }));

const snapshot = (deck: RoomSnapshot['deck']): RoomSnapshot => ({
  deck,
  currentStory: 'Story',
  reveal: false,
  result: null,
  mostAppearingVotes: [],
  mostCommonVotes: [],
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
  roomVersion: null,
});

const renderPanel = (roomSnapshot: RoomSnapshot, onSelectCard = vi.fn()) => {
  render(<RoomVotingPanel
    snapshot={roomSnapshot}
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

  it('does not offer Fibonacci cards before the first room state arrives', () => {
    renderPanel(snapshot(null));

    expect(screen.getByRole('status').textContent).toBe('Loading voting deck...');
    expect(screen.queryByRole('button', { name: '0' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'M' })).toBeNull();
  });

  it('renders and selects cards from the authoritative T-shirt deck', () => {
    const onSelectCard = renderPanel(snapshot(normalizeVotingDeck({
      id: 'tshirt',
      name: 'T-shirt sizes',
      kind: 'categorical',
      cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'],
    })));

    expect(screen.getByText('Deck: T-shirt sizes')).toBeTruthy();
    expect(screen.getAllByRole('button').map((button) => button.textContent)).toEqual(['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕']);
    expect((screen.getByRole('button', { name: 'M' }) as HTMLButtonElement).style.backgroundColor).toBe('rgb(14, 165, 233)');
    fireEvent.click(screen.getByRole('button', { name: 'M' }));
    expect(onSelectCard).toHaveBeenCalledWith('M');
  });
});
