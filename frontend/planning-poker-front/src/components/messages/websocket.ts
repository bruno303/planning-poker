
// WebSocket message types
export type WebSocketMessageType =
  | 'vote'
  | 'reveal-votes'
  | 'new-voting'
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

export interface WebSocketMessage<T = any> {
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
  storyIndex: number;
}

export interface SelectStoryPayload {
  storyId: string;
}

export interface ReorderStoryPayload {
  storyId: string;
  targetIndex: number;
  expectedBacklogVersion: number;
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
  backlogVersion?: number;
}
