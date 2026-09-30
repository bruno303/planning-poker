import type { ConsensusLevel, RoomState, Story } from '@/components/messages/websocket';
import { normalizeMostCommonVotes, normalizeVotingDeck, type VotingDeck } from '@/lib/deck';

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

export type StorySnapshot = Omit<Story, 'mostAppearingVotes' | 'mostCommonVotes'> & {
  mostAppearingVotes: number[];
  mostCommonVotes: string[];
};

export type RoomSnapshot = {
  deck: VotingDeck | null;
  currentStory: string;
  reveal: boolean;
  result: number | null;
  mostAppearingVotes: number[];
  mostCommonVotes: string[];
  consensus: ConsensusLevel | null;
  lowestVote: number | null;
  highestVote: number | null;
  voteRange: number | null;
  voteSpread: number | null;
  nonNumericVoteCount: number;
  participants: Participant[];
  startedAt: string | null;
  backlogMode: boolean;
  stories: StorySnapshot[];
  currentStoryIndex: number;
  roomVersion: number | null;
};

export function normalizeRoomState(state: RoomState): RoomSnapshot {
  return {
    deck: normalizeVotingDeck(state.deck),
    currentStory: state.currentStory,
    reveal: state.reveal,
    result: state.result ?? null,
    mostAppearingVotes: state.mostAppearingVotes ?? [],
    mostCommonVotes: normalizeMostCommonVotes(state.mostCommonVotes, state.mostAppearingVotes),
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
    stories: (state.stories ?? []).map((story) => ({
      ...story,
      mostAppearingVotes: story.mostAppearingVotes ?? [],
      mostCommonVotes: normalizeMostCommonVotes(story.mostCommonVotes, story.mostAppearingVotes),
    })),
    currentStoryIndex: state.currentStoryIndex ?? 0,
    roomVersion: state.roomVersion,
  };
}
