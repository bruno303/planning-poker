
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
  mostAppearingVotes: number[] | null;
  voted: boolean;
}

export interface RoomState {
  type: 'room-state';
  startedAt?: string;
  currentStory: string;
  reveal: boolean;
  result?: number;
  mostAppearingVotes: number[] | null;
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
    votedAt?: string;
    isSpectator: boolean;
    isOwner: boolean;
  }>;
  backlogMode?: boolean;
  stories?: Story[] | null;
  currentStoryIndex?: number;
  roomVersion: number;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isNumberArray(value: unknown): value is number[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'number');
}

function isNullableNumberArray(value: unknown): value is number[] | null {
  return value === null || isNumberArray(value);
}

function isStory(value: unknown): value is Story {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.id === 'string' &&
    typeof value.name === 'string' &&
    isNullableNumberArray(value.mostAppearingVotes) &&
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

const rfc3339Pattern = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;

export function isRFC3339Timestamp(value: unknown): value is string {
  if (typeof value !== 'string') {
    return false;
  }

  const match = rfc3339Pattern.exec(value);
  if (!match) {
    return false;
  }

  const [, year, month, day, hour, minute, second] = match;
  const date = new Date(0);
  date.setUTCFullYear(Number(year), Number(month) - 1, Number(day));
  date.setUTCHours(Number(hour), Number(minute), Number(second), 0);

  return date.getUTCFullYear() === Number(year) &&
    date.getUTCMonth() === Number(month) - 1 &&
    date.getUTCDate() === Number(day) &&
    date.getUTCHours() === Number(hour) &&
    date.getUTCMinutes() === Number(minute) &&
    date.getUTCSeconds() === Number(second) &&
    Number.isFinite(Date.parse(value));
}

function hasOptionalTimestamp(value: Record<string, unknown>, key: string): boolean {
  return !(key in value) || isRFC3339Timestamp(value[key]);
}

export function isRoomState(value: unknown): value is RoomState {
  if (!isRecord(value) || value.type !== 'room-state') {
    return false;
  }

  if (
    typeof value.currentStory !== 'string' ||
    typeof value.reveal !== 'boolean' ||
     !isNullableNumberArray(value.mostAppearingVotes) ||
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
      hasOptionalTimestamp(participant, 'votedAt') &&
      typeof participant.isSpectator === 'boolean' &&
      typeof participant.isOwner === 'boolean'
    );
  }) &&
    hasOptionalTimestamp(value, 'startedAt') &&
    hasOptionalNumber(value, 'result') &&
    (!('consensus' in value) || value.consensus === 'High' || value.consensus === 'Medium' || value.consensus === 'Low' || value.consensus === 'Unavailable') &&
    hasOptionalNumber(value, 'lowestVote') &&
    hasOptionalNumber(value, 'highestVote') &&
    hasOptionalNumber(value, 'voteRange') &&
    hasOptionalNumber(value, 'voteSpread') &&
    hasOptionalNumber(value, 'nonNumericVoteCount') &&
    (!('backlogMode' in value) || typeof value.backlogMode === 'boolean') &&
     (!('stories' in value) || value.stories === null || (Array.isArray(value.stories) && value.stories.every(isStory))) &&
    (!('currentStoryIndex' in value) || typeof value.currentStoryIndex === 'number');
}

export interface StaleCommand {
  type: 'stale-command';
  roomVersion?: number;
}
