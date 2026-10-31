'use client';

import LoadingSpinner from '@/components/loadingSpinner/loadingSpinner';
import type { ClientCommand, GuardedCommandType } from '@/components/messages/websocket';
import { useLogger } from '@/context/logger/loggerContext';
import { useToast } from '@/context/toast/toastContext';
import { useUnvotedReminder } from '@/hooks/useUnvotedReminder';
import { useRoomConnection } from '@/hooks/room/useRoomConnection';
import { useEffect, useRef, useState } from 'react';
import Header from './page.header';
import gridStyles from './page.module.css';
import { styles } from './page.styles';
import RoomParticipantsPanel from './components/roomParticipantsPanel';
import RoomResultsSummary from './components/roomResultsSummary';
import RoomStoryCard from './components/roomStoryCard';
import RoomVotingPanel from './components/roomVotingPanel';
import { useParams, useRouter } from 'next/navigation';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default function PlanningPoker() {
  const logger = useLogger('room-page');
  const params = useParams();
  const router = useRouter();
  const { pushError, pushSuccess } = useToast();
  const routeRoomId = typeof params?.roomId === 'string' ? params.roomId : '';
  const [roomId, setRoomId] = useState('');
  const [userName, setUserName] = useState('');
  const [authorized, setAuthorized] = useState(false);
  const [authorizedRoomId, setAuthorizedRoomId] = useState('');
  const valid = authorized && authorizedRoomId === roomId && roomId === routeRoomId;
  const { snapshot, clientId, isConnected, sendMessage, disconnect } = useRoomConnection({ roomId, userName, enabled: valid });
  const previousSnapshot = useRef(snapshot);
  const [snapshotIdentity, setSnapshotIdentity] = useState(0);
  useEffect(() => {
    if (previousSnapshot.current !== snapshot) {
      previousSnapshot.current = snapshot;
      setSnapshotIdentity((identity) => identity + 1);
    }
  }, [snapshot]);
  const currentUser = snapshot.participants.find((participant) => participant.id === clientId);
  const admin = currentUser?.isOwner ?? false;
  useUnvotedReminder({ participants: snapshot.participants, clientId, currentStory: snapshot.currentStory, isRevealed: snapshot.reveal, isConnected, pushSuccess });

  useEffect(() => {
    if (uuidPattern.test(routeRoomId)) {
      setRoomId(routeRoomId);
      logger.setContext({ roomId: routeRoomId });
      return;
    }

    setRoomId('');
    pushError('Invalid room code. Redirecting to join page.');
    router.replace('/join');
  }, [routeRoomId, router, pushError, logger]);

  useEffect(() => {
    if (!roomId) {
      return;
    }

    setAuthorized(false);
    setAuthorizedRoomId('');
    const name = sessionStorage.getItem('userName');
    if (!name) {
      router.push(`/join/${roomId}`);
      return;
    }

    setUserName(name);
    setAuthorized(true);
    setAuthorizedRoomId(roomId);
    const client = sessionStorage.getItem('clientId');
    if (client) {
      logger.setContext({ clientId: client });
    }

    return () => {
      setAuthorized(false);
      setAuthorizedRoomId('');
    };
  }, [roomId, router, logger]);

  // Guarded commands carry an empty payload here; useRoomConnection injects expectedRoomVersion.
  const sendEmpty = (type: GuardedCommandType) =>
    sendMessage({ type, payload: {} } as unknown as ClientCommand);
  const backHome = () => { disconnect(); sessionStorage.removeItem('clientId'); logger.setContext({ clientId: undefined, roomId: undefined }); router.push('/'); };
  const selectCard = (card: string) => { if (!snapshot.reveal) sendMessage({ type: 'vote', payload: { vote: card } }); };
  if (!valid) {
    return <LoadingSpinner />;
  }

  return (
    <Header
      handleBackToHome={backHome}
      generateShareableLink={() => `${window.location.origin}/room/${roomId}`}
    >
      {!isConnected && (
        <div style={styles.disconnectedBanner} className={gridStyles.disconnectedBanner}>
          Connection lost. Reconnecting...
        </div>
      )}
      <div style={styles.container}>
        <div style={styles.maxWidth}>
          <div style={styles.header}>
            <h1 style={styles.title}>Planning Poker</h1>
            <RoomStoryCard
              currentStory={snapshot.currentStory}
              startedAt={snapshot.startedAt}
              backlogMode={snapshot.backlogMode}
              stories={snapshot.stories}
              currentStoryIndex={snapshot.currentStoryIndex}
              snapshotIdentity={snapshotIdentity}
              amIAdmin={admin}
              onUpdateStory={(story) => sendMessage({ type: 'update-story', payload: { story } })}
              onRemoveStory={(storyId) => sendMessage({ type: 'remove-story', payload: { storyId } })}
              onAddStory={(story) => sendMessage({ type: 'add-story', payload: { story } })}
              onSelectStory={(storyId) => sendMessage({ type: 'select-story', payload: { storyId } })}
              onReorderStory={(storyId, targetIndex) => sendMessage({ type: 'reorder-story', payload: { storyId, targetIndex } })}
              onPreviousStory={() => sendEmpty('prev-story')}
              onNextStory={() => sendEmpty('advance-story')}
            />
          </div>
          <div className={gridStyles.grid}>
            <RoomVotingPanel
              snapshot={snapshot}
              clientId={clientId}
              userName={userName}
              amIAdmin={admin}
              onSelectCard={(card) => { if (card !== null) selectCard(card); }}
              onReveal={() => sendEmpty('reveal-votes')}
              onVoteAgain={() => sendEmpty('vote-again')}
              onToggleBacklog={() => sendEmpty('toggle-backlog-mode')}
            />
            <div style={styles.card}>
              <RoomParticipantsPanel
                snapshot={snapshot}
                amIAdmin={admin}
                onToggleSpectator={(id) => sendMessage({ type: 'toggle-spectator', payload: { targetClientId: id } })}
                onToggleAdmin={(id) => sendMessage({ type: 'toggle-owner', payload: { targetClientId: id } })}
                onCopied={() => pushSuccess('Participant ID copied!')}
              />
              <RoomResultsSummary snapshot={snapshot} />
            </div>
          </div>
        </div>
      </div>
    </Header>
  );
}
