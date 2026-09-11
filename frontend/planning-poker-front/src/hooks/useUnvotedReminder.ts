import { useEffect, useRef } from 'react';
import { playNotificationSound } from '@/lib/notificationSound';

const UNVOTED_NOTIFICATION_MESSAGE = 'The room is waiting for your vote.';
const UNVOTED_NOTIFICATION_DELAY = 4000;
const UNVOTED_NOTIFICATION_INTERVAL = 10000;
const UNVOTED_NOTIFICATION_MAX_REPEATS = 3;

type Card = string | null;

export type ReminderParticipant = {
  id: string;
  name: string;
  vote: Card;
  hasVoted: boolean;
  isSpectator: boolean;
  isOwner: boolean;
};

type UseUnvotedReminderOptions = {
  participants: ReminderParticipant[];
  clientId: string;
  currentStory: string;
  isRevealed: boolean;
  isConnected: boolean;
  pushSuccess: (message: string) => void;
};

export function useUnvotedReminder({
  participants,
  clientId,
  currentStory,
  isRevealed,
  isConnected,
  pushSuccess,
}: UseUnvotedReminderOptions): () => void {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const generationRef = useRef(0);
  const countsRef = useRef(new Map<string, number>());
  const logicalStateRef = useRef('');
  const participantsRef = useRef(participants);
  const clientIdRef = useRef(clientId);
  const currentStoryRef = useRef(currentStory);
  const isRevealedRef = useRef(isRevealed);
  const isConnectedRef = useRef(isConnected);

  participantsRef.current = participants;
  clientIdRef.current = clientId;
  currentStoryRef.current = currentStory;
  isRevealedRef.current = isRevealed;
  isConnectedRef.current = isConnected;

  const cancel = () => {
    generationRef.current++;
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  };

  const logicalState = `${isConnected}:${isRevealed}:${clientId}:${currentStory}:${participants
    .slice()
    .sort((left, right) => left.id.localeCompare(right.id))
    .map((participant) => `${participant.id}:${participant.hasVoted}:${participant.isSpectator}`)
    .join('|')}`;

  useEffect(() => {
    if (logicalStateRef.current === logicalState) return;
    cancel();
    logicalStateRef.current = logicalState;
    countsRef.current.clear();

    const activeParticipants = participants.filter((participant) => !participant.isSpectator);
    const waitingParticipants = activeParticipants.filter((participant) => !participant.hasVoted);
    if (!isConnected || isRevealed || activeParticipants.length < 2 || waitingParticipants.length > 2) {
      return;
    }

    const target = waitingParticipants.length === 1 || activeParticipants.length >= 3
      ? waitingParticipants.find((participant) => participant.id === clientId)
      : undefined;
    if (!target) return;

    const notificationKey = `${logicalState}:${target.id}`;
    const generation = generationRef.current;
    const notify = () => {
      const currentWaiting = participantsRef.current.filter((participant) => !participant.isSpectator && !participant.hasVoted);
      const currentActive = participantsRef.current.filter((participant) => !participant.isSpectator);
    const hasCurrentTarget = currentWaiting.some((participant) => participant.id === clientIdRef.current);
      const currentThreshold = currentWaiting.length === 1 || currentActive.length >= 3;
      if (
        generation !== generationRef.current ||
        logicalStateRef.current !== logicalState ||
        !isConnectedRef.current ||
        isRevealedRef.current ||
        currentStoryRef.current !== currentStory ||
        !currentThreshold ||
        !hasCurrentTarget
      ) {
        return;
      }

      const count = countsRef.current.get(notificationKey) ?? 0;
      if (count >= UNVOTED_NOTIFICATION_MAX_REPEATS) return;
      countsRef.current.set(notificationKey, count + 1);
      pushSuccess(UNVOTED_NOTIFICATION_MESSAGE);
      playNotificationSound();
      if (count + 1 < UNVOTED_NOTIFICATION_MAX_REPEATS) {
        timeoutRef.current = setTimeout(notify, UNVOTED_NOTIFICATION_INTERVAL);
      }
    };

    timeoutRef.current = setTimeout(notify, UNVOTED_NOTIFICATION_DELAY);
  }, [clientId, currentStory, isConnected, isRevealed, logicalState, pushSuccess]);

  useEffect(() => cancel, []);
  return cancel;
}
