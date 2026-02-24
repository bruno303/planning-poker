
// WebSocket message types
export type WebSocketMessageType =
  | 'vote'
  | 'reveal-votes'
  | 'new-voting'
  | 'toggle-spectator'
  | 'toggle-owner'
  | 'vote-again'
  | 'update-name'
  | 'update-story';

export interface WebSocketMessage<T = any> {
  type: WebSocketMessageType;
  payload: T;
}

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
