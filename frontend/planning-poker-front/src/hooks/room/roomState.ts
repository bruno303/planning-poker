import type { ConsensusLevel, RoomState, Story } from '@/components/messages/websocket';

export type Card = string | null;

export type Participant = {
  id: string;
  name: string;
  vote: Card;
  hasVoted: boolean;
  votedAt: string | null;
  isSpectator: boolean;
  isOwner: boolean;
};

export type RoomSnapshot = {
  currentStory: string;
  reveal: boolean;
  result: number | null;
  mostAppearingVotes: string[];
  consensus: ConsensusLevel | null;
  lowestVote: number | null;
  highestVote: number | null;
  voteRange: number | null;
  voteSpread: number | null;
  nonNumericVoteCount: number;
  participants: Participant[];
  startedAt: string | null;
  backlogMode: boolean;
  stories: Story[];
  currentStoryIndex: number;
  roomVersion: number | null;
  deck: string[];
  deckPreset: string | null;
};

export function normalizeRoomState(state: RoomState): RoomSnapshot {
  return {
    currentStory: state.currentStory,
    reveal: state.reveal,
    result: state.result ?? null,
    mostAppearingVotes: (state.mostAppearingVotes ?? []).map(String),
    consensus: state.consensus ?? null,
    lowestVote: state.lowestVote ?? null,
    highestVote: state.highestVote ?? null,
    voteRange: state.voteRange ?? null,
    voteSpread: state.voteSpread ?? null,
    nonNumericVoteCount: state.nonNumericVoteCount ?? 0,
    participants: state.participants.map((participant) => ({
      ...participant,
      votedAt: participant.votedAt ?? null,
    })),
    startedAt: state.startedAt ?? null,
    backlogMode: state.backlogMode ?? false,
    stories: state.stories ?? [],
    currentStoryIndex: state.currentStoryIndex ?? 0,
    roomVersion: state.roomVersion,
    deck: state.deck ?? [],
    deckPreset: state.deckPreset ?? null,
  };
}
