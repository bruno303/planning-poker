import type { ConsensusLevel, DeckDescriptor, RoomState, Story } from '@/components/messages/websocket';

export type Card = string | null;

export const EMPTY_DECK: DeckDescriptor = { id: '', name: '', cards: [] };

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
  deck: DeckDescriptor;
  currentStory: string;
  reveal: boolean;
  result: number | null;
  mostAppearingVotes: string[];
  consensus: ConsensusLevel | null;
  lowestVote: string | null;
  highestVote: string | null;
  voteRange: number | null;
  voteSpread: number | null;
  specialVoteCount: number;
  participants: Participant[];
  startedAt: string | null;
  backlogMode: boolean;
  stories: Story[];
  currentStoryIndex: number;
  roomVersion: number | null;
};

export function normalizeRoomState(state: RoomState): RoomSnapshot {
  return {
    deck: state.deck,
    currentStory: state.currentStory,
    reveal: state.reveal,
    result: state.result ?? null,
    mostAppearingVotes: state.mostAppearingVotes ?? [],
    consensus: state.consensus ?? null,
    lowestVote: state.lowestVote ?? null,
    highestVote: state.highestVote ?? null,
    voteRange: state.voteRange ?? null,
    voteSpread: state.voteSpread ?? null,
    specialVoteCount: state.specialVoteCount ?? 0,
    participants: state.participants.map((participant) => ({
      ...participant,
      votedAt: participant.votedAt ?? null,
    })),
    startedAt: state.startedAt ?? null,
    backlogMode: state.backlogMode ?? false,
    stories: state.stories ?? [],
    currentStoryIndex: state.currentStoryIndex ?? 0,
    roomVersion: state.roomVersion,
  };
}
