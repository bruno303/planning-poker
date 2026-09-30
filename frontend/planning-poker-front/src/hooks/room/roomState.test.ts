import { describe, expect, it } from 'vitest';
import type { RoomState } from '@/components/messages/websocket';
import { normalizeRoomState } from './roomState';

const roomState = (overrides: Partial<RoomState> = {}): RoomState => ({
  type: 'room-state',
  currentStory: 'Estimate a feature',
  reveal: true,
  mostAppearingVotes: [5, 8],
  participants: [],
  roomVersion: 4,
  ...overrides,
});

describe('normalizeRoomState', () => {
  it('defaults a legacy state without a deck descriptor to Fibonacci', () => {
    const normalized = normalizeRoomState(roomState());

    expect(normalized.deck).toMatchObject({ id: 'fibonacci', name: 'Fibonacci', kind: 'numeric' });
    expect(normalized.deck?.cards).toEqual(['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕']);
    expect(normalized.mostCommonVotes).toEqual(['5', '8']);
  });

  it('keeps canonical room and story modes, including explicitly empty arrays', () => {
    const normalized = normalizeRoomState(roomState({
      deck: { id: 'tshirt', name: 'T-shirt sizes', kind: 'categorical', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] },
      mostCommonVotes: [],
      stories: [
        { id: 'categorical-story', name: 'Categorical estimate', mostAppearingVotes: [5], mostCommonVotes: ['M', 'L'], voted: true },
        { id: 'empty-story', name: 'No estimate', mostAppearingVotes: [8], mostCommonVotes: [], voted: true },
      ],
    }));

    expect(normalized.deck?.id).toBe('tshirt');
    expect(normalized.mostCommonVotes).toEqual([]);
    expect(normalized.stories.map((story) => story.mostCommonVotes)).toEqual([['M', 'L'], []]);
  });

  it('converts legacy numeric modes only when the canonical field is absent', () => {
    const normalized = normalizeRoomState(roomState({
      stories: [{ id: 'legacy-story', name: 'Legacy estimate', mostAppearingVotes: [3, 5], voted: true }],
    }));

    expect(normalized.stories[0].mostCommonVotes).toEqual(['3', '5']);
    expect(normalized.mostAppearingVotes).toEqual([5, 8]);
  });
});
