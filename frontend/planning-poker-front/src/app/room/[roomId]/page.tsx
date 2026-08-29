'use client'


import BacklogModal from '@/components/backlogModal/backlogModal';
import Avatar from '@/components/avatar/avatar';
import { getExtremeVotes } from '@/components/consensus/consensus';
import FocusableComponent from '@/components/focusableInput/focusableInput';
import LoadingSpinner from '@/components/loadingSpinner/loadingSpinner';
import {
  AddStoryPayload,
  ConsensusLevel,
  ReorderStoryPayload,
  RemoveStoryPayload,
  RoomState,
  SelectStoryPayload,
  Story,
  ToggleOwnerPayload,
  ToggleSpectatorPayload,
  UpdateNamePayload,
  UpdateStoryPayload,
  VotePayload,
  WebSocketMessage
} from '@/components/messages/websocket';
import ParticipantIdBadge from '@/components/participantIdBadge/participantIdBadge';
import { useLogger } from '@/context/logger/loggerContext';
import { useRoom } from '@/context/room/roomContext';
import { useToast } from '@/context/toast/toastContext';
import { ChevronLeft, ChevronRight, Eye, EyeOff, List, Repeat, RotateCcw, Shield, Users, X } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import Header from './page.header';
import gridStyles from './page.module.css';
import { styles } from './page.styles';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

const isValidRoomId = (value: string): boolean => uuidPattern.test(value);

const RECONNECT_INITIAL_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;
const RECONNECT_MULTIPLIER = 2;

type Card = string | null

type Participant = {
  id: string
  name: string
  vote: Card
  hasVoted: boolean
  isSpectator: boolean
  isOwner: boolean
}

export default function PlanningPoker() {
  const logger = useLogger('room-page');
  const params = useParams();
  const router = useRouter();
  const { socket, connected } = useRoom();
  const { pushError, pushSuccess } = useToast();
  const routeRoomId = typeof params?.roomId === 'string' ? params.roomId : '';
  const [roomId, setRoomId] = useState('');
  const connectedRoomIdRef = useRef<string | null>(null);

  const [selectedCard, setSelectedCard] = useState<Card>(null);
  const [userName, setUserName] = useState('');
  const [isRevealed, setIsRevealed] = useState(false);
  const [currentStory, setCurrentStory] = useState('');
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [clientId, setClientId] = useState('');
  const [result, setResult] = useState<number | null>(null);
  const [mostAppearingVotes, setMostAppearingVotes] = useState<number[]>([]);
  const [consensus, setConsensus] = useState<ConsensusLevel | null>(null);
  const [lowestVote, setLowestVote] = useState<number | null>(null);
  const [highestVote, setHighestVote] = useState<number | null>(null);
  const [voteRange, setVoteRange] = useState<number | null>(null);
  const [voteSpread, setVoteSpread] = useState<number | null>(null);
  const [nonNumericVoteCount, setNonNumericVoteCount] = useState(0);
  const [isEditingStory, setIsEditingStory] = useState(false);
  const [isConnected, setIsConnected] = useState(true);
  const [isAuthorized, setIsAuthorized] = useState(false);
  const [backlogMode, setBacklogMode] = useState(false);
  const [stories, setStories] = useState<Story[]>([]);
  const [currentStoryIndex, setCurrentStoryIndex] = useState(0);
  const [showBacklogModal, setShowBacklogModal] = useState(false);
  const deliberateDisconnect = useRef(false);
  const editingStoryRef = useRef('');
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);

  // Planning poker cards (Fibonacci sequence + special cards)
  const cards = ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'];

  useEffect(() => {
    if (isValidRoomId(routeRoomId)) {
      setRoomId(routeRoomId);
      logger.setContext({ roomId: routeRoomId });
      return;
    }

    setRoomId('');
    logger.info('Invalid room ID', { roomId: routeRoomId });
    pushError('Invalid room code. Redirecting to join page.');
    router.replace('/join');
  }, [routeRoomId, router, pushError, logger]);

  useEffect(() => {
    if (!roomId) {
      return;
    }

    const storedUserName = sessionStorage.getItem('userName');
    if (!storedUserName) {
      router.push(`/join/${roomId}`);
      return;
    }
    setUserName(storedUserName);
    setIsAuthorized(true);

    const storedClientId = sessionStorage.getItem('clientId');
    if (storedClientId) {
      logger.setContext({ clientId: storedClientId });
    }

    const hasSameRoomConnection =
      connected.current &&
      connectedRoomIdRef.current === roomId &&
      socket.current?.readyState !== WebSocket.CLOSED;

    if (hasSameRoomConnection) {
      return;
    }

    cleanupSocket();
    connectWebSocket(roomId, storedUserName);

    return () => {
      setIsAuthorized(false);
      cancelReconnect();
      if (connectedRoomIdRef.current === roomId) {
        cleanupSocket();
      }
    };
  }, [roomId, router, logger]);

  useEffect(() => {
    if (clientId && participants.length > 0) {
      setSelectedCard(getCurrentUser()?.vote ?? null);
    }
  }, [participants, clientId]);

  const sendMessage = <T,>(message: WebSocketMessage<T>) => {
    const activeSocket = socket.current;

    if (!activeSocket || activeSocket.readyState !== WebSocket.OPEN) {
      pushError('Connection is not ready. Please wait and try again.');
      return;
    }

    try {
      activeSocket.send(JSON.stringify(message));
    } catch (err: any) {
      const errorMessage = err?.message
        ? `Failed to send message: ${err.message}`
        : 'Failed to send message.';
      pushError(errorMessage);
    }
  }

  const handleCardSelect = (card: Card) => {
    if (!isRevealed) {
      const payload: VotePayload = { vote: card };
      sendMessage<VotePayload>({ type: 'vote', payload });
    }
  };

  const handleRevealVotes = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'reveal-votes', payload });
  };

  const handleNewVoting = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'new-voting', payload });
  };

  const handleToggleSpectator = (participantId: string) => {
    const payload: ToggleSpectatorPayload = { targetClientId: participantId };
    sendMessage<ToggleSpectatorPayload>({ type: 'toggle-spectator', payload });
  };

  const handleToggleAdmin = (participantId: string) => {
    const payload: ToggleOwnerPayload = { targetClientId: participantId };
    sendMessage<ToggleOwnerPayload>({ type: 'toggle-owner', payload });
  };

  const handleVoteAgain = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'vote-again', payload });
  }

  const handleToggleBacklogMode = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'toggle-backlog-mode', payload });
  };

  const handleAddStory = (story: string) => {
    const trimmed = story.trim();
    if (!trimmed) return;
    const payload: AddStoryPayload = { story: trimmed };
    sendMessage<AddStoryPayload>({ type: 'add-story', payload });
  };

  const handleRemoveStory = (storyId: string) => {
    const payload: RemoveStoryPayload = { storyId };
    sendMessage<RemoveStoryPayload>({ type: 'remove-story', payload });
  };

  const handleSelectStory = (storyId: string) => {
    const payload: SelectStoryPayload = { storyId };
    sendMessage<SelectStoryPayload>({ type: 'select-story', payload });
  };

  const handleReorderStory = (storyId: string, targetIndex: number) => {
    const payload: ReorderStoryPayload = { storyId, targetIndex };
    sendMessage<ReorderStoryPayload>({ type: 'reorder-story', payload });
  };

  const handleStartStoryEdit = () => {
    editingStoryRef.current = currentStory;
    setIsEditingStory(true);
  };

  const handleCancelStoryEdit = () => {
    setCurrentStory(editingStoryRef.current);
    setIsEditingStory(false);
  };

  const handleSaveStoryEdit = () => {
    const trimmedStory = currentStory.trim();

    if (!trimmedStory && backlogMode && stories.length > 0) {
      const shouldRemoveStory = window.confirm('The task name is empty. Do you want to remove this task?');
      if (!shouldRemoveStory) {
        return;
      }
      const currentStoryToRemove = stories[currentStoryIndex];
      if (currentStoryToRemove) {
        handleRemoveStory(currentStoryToRemove.id);
      }
    } else {
      const payload: UpdateStoryPayload = { story: trimmedStory };
      sendMessage<UpdateStoryPayload>({ type: 'update-story', payload });
    }

    setIsEditingStory(false);
  };

  const handleAdvanceStory = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'advance-story', payload });
  };

  const handlePrevStory = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'prev-story', payload });
  };

  const cancelReconnect = () => {
    if (reconnectTimeoutRef.current !== null) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  };

  const scheduleReconnect = (roomCode: string, storedUserName: string) => {
    const delay = Math.min(
      RECONNECT_INITIAL_DELAY * Math.pow(RECONNECT_MULTIPLIER, reconnectAttemptsRef.current),
      RECONNECT_MAX_DELAY
    );
    reconnectAttemptsRef.current++;
    logger.warn('Reconnection attempt', { attempt: reconnectAttemptsRef.current });
    reconnectTimeoutRef.current = setTimeout(() => {
      connectWebSocket(roomCode, storedUserName);
    }, delay);
  };

  const cleanupSocket = () => {
    deliberateDisconnect.current = true;
    cancelReconnect();
    const activeSocket = socket.current;
    if (activeSocket) {
      activeSocket.onopen = null;
      activeSocket.onmessage = null;
      activeSocket.onclose = null;
      activeSocket.onerror = null;
      activeSocket.close();
    }
    connected.current = false;
    connectedRoomIdRef.current = null;
    socket.current = null;
  };

  const connectWebSocket = (roomCode: string, userName: string) => {
    deliberateDisconnect.current = false;
    const savedClientId = sessionStorage.getItem('clientId');
    const wsUrl = savedClientId
      ? `${process.env.NEXT_PUBLIC_WEBSOCKET_URL}/planning/${roomCode}/ws?clientId=${encodeURIComponent(savedClientId)}`
      : `${process.env.NEXT_PUBLIC_WEBSOCKET_URL}/planning/${roomCode}/ws`;
    const ws = new WebSocket(wsUrl);
    socket.current = ws;
    if (process.env.NODE_ENV !== 'production' || process.env.NEXT_PUBLIC_EXPOSE_WS_GLOBAL === 'true') {
      (window as any).__ws = ws;
    }
    connected.current = true;
    connectedRoomIdRef.current = roomCode;

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        if (data.type === 'room-state') {
          const roomState = data as RoomState;
          setParticipants(roomState.participants);
          setCurrentStory(roomState.currentStory);
          setIsRevealed(roomState.reveal);
          setSelectedCard(getCurrentUser()?.vote ?? null);
          setResult(roomState.result ?? null);
          setMostAppearingVotes(roomState.mostAppearingVotes ?? []);
          setConsensus(roomState.consensus ?? null);
          setLowestVote(roomState.lowestVote ?? null);
          setHighestVote(roomState.highestVote ?? null);
          setVoteRange(roomState.voteRange ?? null);
          setVoteSpread(roomState.voteSpread ?? null);
          setNonNumericVoteCount(roomState.nonNumericVoteCount ?? 0);
          setBacklogMode(roomState.backlogMode ?? false);
          setStories(roomState.stories ?? []);
          setCurrentStoryIndex(roomState.currentStoryIndex ?? 0);

        } else if (data.type === 'update-client-id') {
          setClientId(data.clientId);
          sessionStorage.setItem('clientId', data.clientId);
          logger.setContext({ clientId: data.clientId });
          const payload: UpdateNamePayload = { username: userName };
          sendMessage<UpdateNamePayload>({ type: 'update-name', payload });

        } else if (data.type === 'kicked') {
          deliberateDisconnect.current = true;
          cancelReconnect();
          logger.info('Kicked from room');
          sessionStorage.removeItem('clientId');
          logger.setContext({ clientId: undefined, roomId: undefined });
          pushError('You have been kicked from the room');
          router.push('/');

        } else {
          throw new Error('Invalid message from websocket');
        }
      } catch (err: any) {
        logger.error('Message handling failed', { error: err?.message });
        const message = err?.message ? `Error while handling websocket message: ${err.message}` : 'Error while handling websocket message';
        pushError(message);
      }
    };

    ws.onopen = () => {
      logger.info('WebSocket connected');
      setIsConnected(true);
      reconnectAttemptsRef.current = 0;
      pushSuccess('Connected');
    };

    ws.onclose = (event) => {
      logger.warn('WebSocket closed', { code: event.code, reason: event.reason });
      setIsConnected(false);
      if (socket.current === ws) {
        socket.current = null;
        connected.current = false;
        connectedRoomIdRef.current = null
      }
      if (!deliberateDisconnect.current) {
        scheduleReconnect(roomCode, userName);
      }
    };

    ws.onerror = (event) => {
      logger.error('WebSocket error', { error: event.type });
      pushError('Connection error');
      if (socket.current === ws) {
        connected.current = false;
      }
    };
  };

  const getCurrentUser = () => {
    return participants.find((p) => p.id === clientId);
  }

  const isAdmin = (): boolean => {
    return getCurrentUser()?.isOwner ?? false
  }

  const getCardColor = (card: Card) => {
    if (card === '?') return '#8b5cf6'; // purple
    if (card === '☕') return '#f59e0b'; // amber
    const num = parseInt(card ?? '');
    if (num <= 2) return '#10b981'; // green
    if (num <= 8) return '#eab308'; // yellow
    if (num <= 21) return '#f97316'; // orange
    return '#ef4444'; // red
  };

  const handleBackToHome = () => {
    cleanupSocket();
    sessionStorage.removeItem('clientId');
    logger.setContext({ clientId: undefined, roomId: undefined });
    router.push('/');
  };

  const votedCount = participants.filter(p => !p.isSpectator && p.hasVoted).length;
  const totalVoters = participants.filter(p => !p.isSpectator).length;

  const amIAdmin = isAdmin();

  const canGoPrev = currentStoryIndex > 0;
  const canGoNext = stories.length > 0 && currentStoryIndex < stories.length - 1;

  const getParticipantExtremes = (participant: Participant) => {
    if (!isRevealed || participant.isSpectator) {
      return [];
    }

    return getExtremeVotes(participant.vote, lowestVote, highestVote);
  };

  if (!isAuthorized) {
    return <LoadingSpinner />;
  }

  return (

    <Header
      handleBackToHome={handleBackToHome}
      generateShareableLink={() => `${window.location.origin}/room/${roomId}`}
    >
      {!isConnected && (
        <div style={styles.disconnectedBanner} className={gridStyles.disconnectedBanner}>
          Connection lost. Reconnecting...
        </div>
      )}
      <div style={styles.container}>
        <div style={styles.maxWidth}>
          {/* Header */}
          <div style={styles.header}>
            <h1 style={styles.title}>Planning Poker</h1>

            <div style={styles.storyCard}>
              <div style={styles.storyHeader}>
                <h2 style={styles.storyTitle}>Current Story</h2>
                {amIAdmin && !isEditingStory && ((backlogMode && currentStory) || !backlogMode) && (
                  <button
                    style={{ ...styles.button, ...styles.primaryButton, ...styles.storyEditButton }}
                    onClick={handleStartStoryEdit}
                  >
                    Edit
                  </button>
                )}
              </div>
              <div style={styles.storyLine}>
                {amIAdmin ? (
                  <div style={styles.storyContent}>
                    {isEditingStory ? (
                      <>
                        <FocusableComponent
                          currentStory={currentStory}
                          onChange={e => setCurrentStory(e.target.value)}
                          onKeyDown={e => {
                            if (e.key === 'Enter') {
                              handleSaveStoryEdit();
                            } else if (e.key === 'Escape') {
                              handleCancelStoryEdit();
                            }
                          }}
                        />
                        <button
                          style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                          onClick={handleSaveStoryEdit}
                        >
                          Save
                        </button>
                      </>
                    ) : (
                      <label style={{ ...styles.label, margin: 0, flex: 1, textAlign: 'center' }}>
                        {currentStory}
                        {backlogMode && stories.length > 0 && (
                          <span style={styles.backlogStoryPosition}>
                            (Story {currentStoryIndex + 1} of {stories.length})
                          </span>
                        )}
                      </label>
                    )}
                  </div>
                ) : (
                  <p
                    style={{
                      ...styles.storyText,
                      margin: 0,
                      width: '100%',
                      gridColumn: 2,
                      textAlign: 'center'
                    }}
                  >
                    {currentStory}
                  </p>
                )}

                <div style={styles.storyControls}>
                  {backlogMode && amIAdmin && (
                    <div style={styles.storyNavigation}>
                      <button
                        onClick={handlePrevStory}
                        disabled={!canGoPrev}
                        aria-label="Previous Story"
                        title="Previous Story"
                        style={{
                          ...styles.storyControlButton,
                          ...(!canGoPrev ? styles.buttonDisabled : {})
                        }}
                      >
                        <ChevronLeft size={20} aria-hidden="true" />
                      </button>
                      <button
                        onClick={handleAdvanceStory}
                        disabled={!canGoNext}
                        aria-label="Next Story"
                        title="Next Story"
                        style={{
                          ...styles.storyControlButton,
                          ...(!canGoNext ? styles.buttonDisabled : {})
                        }}
                      >
                        <ChevronRight size={20} aria-hidden="true" />
                      </button>
                    </div>
                  )}
                  {backlogMode && (
                    <button
                      style={{ ...styles.button, ...styles.primaryButton, ...styles.storyBacklogButton }}
                      aria-label="Open backlog"
                      onClick={() => setShowBacklogModal(true)}
                    >
                      <List size={18} aria-hidden="true" />
                      Backlog
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div className={gridStyles.grid}>
            {/* Main Voting Area */}
            <div style={styles.card}>
              {/* User Info */}
              <div style={styles.userInfo}>
                <div style={styles.inputGroup}>
                  {/* <label style={styles.label}>Your Name</label> */}
                  <div style={styles.userNameRow}>
                    <Avatar participant={{ id: clientId }} />
                    <label style={{ ...styles.label, marginBottom: 0 }}>{userName}</label>
                  </div>
                </div>
                <div style={styles.voteStats}>
                  <div style={styles.voteStatsLabel}>Votes Cast</div>
                  <div style={styles.voteStatsNumber}>{votedCount}/{totalVoters}</div>
                </div>
              </div>

              <div style={styles.selectedCard}>
                <div style={styles.selectedCardLabel}>
                  {selectedCard ? 'Your Vote' : 'No Vote Yet'}
                </div>
                <div style={{
                  ...styles.selectedCardDisplay,
                  backgroundColor: selectedCard ? getCardColor(selectedCard) : '#9ca3af'
                }}>
                  {selectedCard ? selectedCard : <X size={32} strokeWidth={3} />}
                </div>
              </div>

              {/* Planning Poker Cards */}
              <div style={{ marginTop: '2rem' }}>
                <h3 style={styles.sectionTitle}>Select Your Card</h3>
                <div style={styles.cardsGrid}>
                  {cards.map((card) => (
                    <button
                      key={card}
                      onClick={() => !isRevealed && handleCardSelect(card)}
                      aria-disabled={isRevealed}
                      aria-pressed={selectedCard === card}
                      style={{
                        ...styles.pokerCard,
                        ...(isRevealed ? styles.pokerCardDisabled : {}),
                        backgroundColor: isRevealed ? '#9ca3af' : getCardColor(card),
                        ...(selectedCard === card ? styles.pokerCardSelected : {})
                      }}
                      onMouseEnter={(e) => {
                        if (!isRevealed && selectedCard !== card) {
                          (e.target as HTMLButtonElement).style.transform = 'scale(1.05)';
                        }
                      }}
                      onMouseLeave={(e) => {
                        if (!isRevealed && selectedCard !== card) {
                          (e.target as HTMLButtonElement).style.transform = 'scale(1)';
                        }
                      }}
                    >
                      {card}
                    </button>
                  ))}
                </div>
              </div>

              {/* Action Buttons */}
              {amIAdmin && (
                <div style={styles.buttonsContainer}>
                  {!backlogMode && (
                    <button
                      onClick={handleToggleBacklogMode}
                      style={{ ...styles.button, ...styles.primaryButton }}
                      onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#2563eb'}
                      onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}
                    >
                      <List size={20} />
                      Enable Backlog
                    </button>
                  )}
                  <button
                    onClick={handleRevealVotes}
                    style={{ ...styles.button, ...styles.primaryButton }}
                    onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#2563eb'}
                    onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}
                  >
                    {isRevealed ? <EyeOff size={20} /> : <Eye size={20} />}
                    {isRevealed ? 'Hide Votes' : 'Reveal Votes'}
                  </button>
                  <button
                    onClick={handleNewVoting}
                    style={{ ...styles.button, ...styles.successButton }}
                    onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#059669'}
                    onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#10b981'}
                  >
                    <RotateCcw size={20} />
                    New Voting
                  </button>
                  <button
                    onClick={handleVoteAgain}
                    style={{ ...styles.button, ...styles.warningButton }}
                    onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#f97316'}
                    onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#eab308'}
                  >
                    <Repeat size={20} />
                    Vote Again
                  </button>
                </div>
              )}
            </div>

            {/* Participants Panel */}
            <div style={styles.card}>
              <div style={styles.participantsHeader}>
                <Users color="#3b82f6" size={24} />
                <h3 style={styles.sectionTitle}>Participants</h3>
              </div>

              <div style={styles.participantsList}>
                {participants.map((participant) => (
                  <div
                    key={participant.id}
                    style={{
                      ...styles.participant,
                      ...(participant.isSpectator
                        ? styles.participantSpectator
                        : participant.hasVoted
                          ? styles.participantVoted
                          : styles.participantWaiting),
                      ...(getParticipantExtremes(participant).includes('lowest')
                        ? styles.participantLowest
                        : getParticipantExtremes(participant).includes('highest')
                          ? styles.participantHighest
                          : {})
                    }}
                  >
                    <div style={styles.participantContent}>
                      <div>
                        <div style={styles.participantName}>
                          <Avatar participant={participant} />
                          {participant.name}
                          {amIAdmin && (
                            <ParticipantIdBadge
                              participantId={participant.id}
                              onCopied={() => pushSuccess('Participant ID copied!')}
                            />
                          )}
                          {getParticipantExtremes(participant).map((extreme) => (
                            <span
                              key={extreme}
                              style={extreme === 'lowest' ? styles.lowestVoteBadge : styles.highestVoteBadge}
                              title={`${extreme === 'lowest' ? 'Lowest' : 'Highest'} numeric estimate`}
                            >
                              {extreme === 'lowest' ? 'Lowest' : 'Highest'}
                            </span>
                          ))}
                        </div>
                        <div style={styles.participantStatus}>
                          {participant.isSpectator ? 'Spectator' : participant.hasVoted ? 'Voted' : 'Waiting...'}
                        </div>
                      </div>
                      <div style={styles.participantRight}>
                        {!participant.isSpectator && participant.hasVoted && (
                          <div style={{
                            ...styles.voteCard,
                            backgroundColor: isRevealed ? getCardColor(participant.vote) : '#9ca3af'
                          }}>
                            {isRevealed ? participant.vote : '?'}
                          </div>
                        )}

                        {amIAdmin && (
                          <div style={styles.adminControls}>
                            <button
                              onClick={() => handleToggleSpectator(participant.id)}
                              style={{
                                ...styles.roleButton,
                                ...(participant.isSpectator ? styles.activeSpectatorButton : styles.inactiveButton)
                              }}
                              title={participant.isSpectator ? 'Make Voter' : 'Make Spectator'}
                            >
                              <Eye size={12} />
                            </button>
                            <button
                              onClick={() => handleToggleAdmin(participant.id)}
                              style={{
                                ...styles.roleButton,
                                ...(participant.isOwner ? styles.activeAdminButton : styles.inactiveButton)
                              }}
                              title={participant.isOwner ? 'Remove Admin' : 'Make Admin'}
                            >
                              <Shield size={12} />
                            </button>
                          </div>
                        )}

                        <div style={{
                          ...styles.statusDot,
                          ...(participant.isSpectator
                            ? styles.statusSpectator
                            : participant.hasVoted
                              ? styles.statusVoted
                              : styles.statusWaiting)
                        }} />
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              {/* Summary */}
              {isRevealed && (
                <div style={styles.summary}>
                  <h4 style={styles.summaryTitle}>Results Summary</h4>
                  <div style={styles.summaryContent}>
                    <div>Consensus: {consensus ?? 'Unavailable'}</div>
                    <div>Average: {result !== null ? result.toFixed(1) : 'Unavailable'}</div>
                    <div>Most Common: {mostAppearingVotes.length > 0 ? mostAppearingVotes.join(", ") : 'Unavailable'}</div>
                    {lowestVote !== null && highestVote !== null && (
                      <div>Votes range from {lowestVote} to {highestVote} (spread: {voteRange ?? 0})</div>
                    )}
                    {voteSpread !== null && <div>Deck spread: {voteSpread} step{voteSpread === 1 ? '' : 's'}</div>}
                    {nonNumericVoteCount > 0 && (
                      <div>
                        Non-numeric votes: {nonNumericVoteCount}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Backlog Modal */}
          {showBacklogModal && (
            <BacklogModal
              stories={stories}
              currentStoryIndex={currentStoryIndex}
              amIAdmin={amIAdmin}
              onClose={() => setShowBacklogModal(false)}
              onAddStory={handleAddStory}
              onRemoveStory={handleRemoveStory}
              onSelectStory={handleSelectStory}
              onReorderStory={handleReorderStory}
            />
          )}
        </div>
      </div>
    </Header>
  );
}
