
// WebSocket message types
export type WebSocketMessageType =
  | 'vote'
  | 'reveal-votes'
  | 'toggle-spectator'
  | 'toggle-owner'
  | 'vote-again'
  | 'update-name'
  | 'update-story'
  | 'kicked'
  | 'toggle-backlog-mode'
  | 'add-story'
  | 'remove-story'
  | 'advance-story'
  | 'prev-story'
  | 'select-story'
  | 'reorder-story';

export interface WebSocketMessage<T = unknown> {
  type: WebSocketMessageType;
  payload: T;
}

export type ConsensusLevel = 'High' | 'Medium' | 'Low' | 'Unavailable';

export interface VotePayload {
  vote: string | null;
}

export interface ToggleSpectatorPayload {
  targetClientId: string;
}

export interface ToggleOwnerPayload {
  targetClientId: string;
}

export interface UpdateNamePayload {
  username: string;
}

export interface UpdateStoryPayload {
  story: string;
}

export interface AddStoryPayload {
  story: string;
}

export interface RemoveStoryPayload {
  storyId: string;
}

export interface SelectStoryPayload {
  storyId: string;
}

export interface ReorderStoryPayload {
  storyId: string;
  targetIndex: number;
}

export interface Story {
  id: string;
  name: string;
  result?: number;
  mostAppearingVotes: number[];
  voted: boolean;
}

export interface RoomState {
  type: 'room-state';
  currentStory: string;
  reveal: boolean;
  result?: number;
  mostAppearingVotes: number[];
  consensus?: ConsensusLevel;
  lowestVote?: number;
  highestVote?: number;
  voteRange?: number;
  voteSpread?: number;
  nonNumericVoteCount?: number;
  participants: Array<{
    id: string;
    name: string;
    vote: string | null;
    hasVoted: boolean;
    isSpectator: boolean;
    isOwner: boolean;
  }>;
  backlogMode?: boolean;
  stories?: Story[];
  currentStoryIndex?: number;
  roomVersion: number;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isNumberArray(value: unknown): value is number[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'number');
}

function isStory(value: unknown): value is Story {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.id === 'string' &&
    typeof value.name === 'string' &&
    isNumberArray(value.mostAppearingVotes) &&
    typeof value.voted === 'boolean' &&
    (!('result' in value) || typeof value.result === 'number')
  );
}

function hasOptionalNumber(value: Record<string, unknown>, key: string): boolean {
  return !(
    key in value &&
    typeof value[key] !== 'number'
  );
}

export function isRoomState(value: unknown): value is RoomState {
  if (!isRecord(value) || value.type !== 'room-state') {
    return false;
  }

  if (
    typeof value.currentStory !== 'string' ||
    typeof value.reveal !== 'boolean' ||
    !isNumberArray(value.mostAppearingVotes) ||
    typeof value.roomVersion !== 'number' ||
    !Array.isArray(value.participants)
  ) {
    return false;
  }

  return value.participants.every((participant) => {
    if (!isRecord(participant)) {
      return false;
    }

    return (
      typeof participant.id === 'string' &&
      typeof participant.name === 'string' &&
      (typeof participant.vote === 'string' || participant.vote === null) &&
      typeof participant.hasVoted === 'boolean' &&
      typeof participant.isSpectator === 'boolean' &&
      typeof participant.isOwner === 'boolean'
    );
  }) &&
    hasOptionalNumber(value, 'result') &&
    (!('consensus' in value) || value.consensus === 'High' || value.consensus === 'Medium' || value.consensus === 'Low' || value.consensus === 'Unavailable') &&
    hasOptionalNumber(value, 'lowestVote') &&
    hasOptionalNumber(value, 'highestVote') &&
    hasOptionalNumber(value, 'voteRange') &&
    hasOptionalNumber(value, 'voteSpread') &&
    hasOptionalNumber(value, 'nonNumericVoteCount') &&
    (!('backlogMode' in value) || typeof value.backlogMode === 'boolean') &&
    (!('stories' in value) || (Array.isArray(value.stories) && value.stories.every(isStory))) &&
    (!('currentStoryIndex' in value) || typeof value.currentStoryIndex === 'number');
}

export interface StaleCommand {
  type: 'stale-command';
  roomVersion?: number;
}
