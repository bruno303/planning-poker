import { describe, expect, it } from 'vitest';

import { isRFC3339Timestamp, isRoomState, type RoomState } from './websocket';

const validRoomState: RoomState = {
  type: 'room-state',
  deck: { id: 'fibonacci', name: 'Fibonacci', kind: 'numeric', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] },
  startedAt: '2026-09-03T12:00:00Z',
  currentStory: 'Estimate the story',
  reveal: true,
  result: 5,
  mostCommonVotes: ['5', '8'],
  mostAppearingVotes: [5, 8],
  consensus: 'Medium',
  lowestVote: 3,
  highestVote: 8,
  voteRange: 5,
  voteSpread: 2,
  nonNumericVoteCount: 0,
  participants: [{
    id: 'participant-1',
    name: 'Ada',
    vote: '5',
    hasVoted: true,
    votedAt: '2026-09-03T12:01:00.250Z',
    isSpectator: false,
    isOwner: true,
  }],
  backlogMode: true,
  stories: [{
    id: 'story-1',
    name: 'A backlog story',
    result: 8,
    mostCommonVotes: ['8'],
    mostAppearingVotes: [8],
    voted: true,
  }],
  currentStoryIndex: 0,
  roomVersion: 4,
};

describe('isRoomState', () => {
  it('accepts a complete room state', () => {
    expect(isRoomState(validRoomState)).toBe(true);
  });

  it('accepts null slice values emitted for empty backend collections', () => {
    expect(isRoomState({
      ...validRoomState,
      mostAppearingVotes: null,
      stories: null,
    })).toBe(true);
  });

  it('accepts legacy states without timestamp fields', () => {
    const legacyState: RoomState = {
      ...validRoomState,
      participants: validRoomState.participants.map((participant) => ({ ...participant })),
    };
    delete legacyState.startedAt;
    delete legacyState.participants[0].votedAt;

    expect(isRoomState(legacyState)).toBe(true);
  });

  it('accepts legacy states without deck descriptors or string modes', () => {
    const legacyState: RoomState = {
      ...validRoomState,
      participants: validRoomState.participants.map((participant) => ({ ...participant })),
      stories: validRoomState.stories?.map((story) => ({ ...story })),
    };
    delete legacyState.deck;
    delete legacyState.mostCommonVotes;
    delete legacyState.stories?.[0].mostCommonVotes;

    expect(isRoomState(legacyState)).toBe(true);
  });

  it.each([
    ['required field', 'currentStory', null],
    ['vote array member', 'mostAppearingVotes', ['5']],
    ['string mode member', 'mostCommonVotes', ['M', 8]],
    ['deck descriptor', 'deck', { id: 'unknown', name: 'Unknown', kind: 'categorical', cards: ['M'] }],
    ['participant member', 'participants', [{ ...validRoomState.participants[0], hasVoted: 'yes' }]],
    ['optional number', 'result', '5'],
    ['optional consensus', 'consensus', 'Unknown'],
    ['optional boolean', 'backlogMode', 'true'],
    ['optional story list', 'stories', [{}]],
    ['optional story member', 'stories', [{ ...validRoomState.stories?.[0], voted: 'yes' }]],
    ['optional story result', 'stories', [{ ...validRoomState.stories?.[0], result: null }]],
    ['optional story votes', 'stories', [{ ...validRoomState.stories?.[0], mostAppearingVotes: [null] }]],
    ['optional story string votes', 'stories', [{ ...validRoomState.stories?.[0], mostCommonVotes: [5] }]],
    ['optional story index', 'currentStoryIndex', '0'],
    ['optional room timestamp', 'startedAt', 'not-a-timestamp'],
    ['optional participant timestamp', 'participants', [{ ...validRoomState.participants[0], votedAt: 'not-a-timestamp' }]],
  ])('rejects malformed %s', (_, field, replacement) => {
    const malformed = { ...validRoomState, [field]: replacement };

    expect(isRoomState(malformed)).toBe(false);
  });

  it('rejects a present optional field with undefined', () => {
    expect(isRoomState({ ...validRoomState, result: undefined })).toBe(false);
  });

  it('validates RFC3339 timestamp values', () => {
    expect(isRFC3339Timestamp('2026-09-03T12:00:00Z')).toBe(true);
    expect(isRFC3339Timestamp('2026-09-03T12:00:00.250-03:00')).toBe(true);
    expect(isRFC3339Timestamp('2026-09-03 12:00:00Z')).toBe(false);
    expect(isRFC3339Timestamp('2026-02-30T12:00:00Z')).toBe(false);
  });
});
