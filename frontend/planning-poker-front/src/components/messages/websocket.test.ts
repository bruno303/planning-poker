import { describe, expect, it } from 'vitest';

import { isRoomState, type RoomState } from './websocket';

const validRoomState: RoomState = {
  type: 'room-state',
  currentStory: 'Estimate the story',
  reveal: true,
  result: 5,
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
    isSpectator: false,
    isOwner: true,
  }],
  backlogMode: true,
  stories: [{
    id: 'story-1',
    name: 'A backlog story',
    result: 8,
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

  it.each([
    ['required field', 'currentStory', null],
    ['vote array member', 'mostAppearingVotes', ['5']],
    ['participant member', 'participants', [{ ...validRoomState.participants[0], hasVoted: 'yes' }]],
    ['optional number', 'result', '5'],
    ['optional consensus', 'consensus', 'Unknown'],
    ['optional boolean', 'backlogMode', 'true'],
    ['optional story list', 'stories', [{}]],
    ['optional story member', 'stories', [{ ...validRoomState.stories?.[0], voted: 'yes' }]],
    ['optional story result', 'stories', [{ ...validRoomState.stories?.[0], result: null }]],
    ['optional story votes', 'stories', [{ ...validRoomState.stories?.[0], mostAppearingVotes: [null] }]],
    ['optional story index', 'currentStoryIndex', '0'],
  ])('rejects malformed %s', (_, field, replacement) => {
    const malformed = { ...validRoomState, [field]: replacement };

    expect(isRoomState(malformed)).toBe(false);
  });

  it('rejects a present optional field with undefined', () => {
    expect(isRoomState({ ...validRoomState, result: undefined })).toBe(false);
  });
});
