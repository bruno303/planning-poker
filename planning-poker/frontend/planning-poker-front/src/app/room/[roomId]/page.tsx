'use client'


import FocusableComponent from '@/components/focusableInput/focusableInput';
import {
  ToggleOwnerPayload,
  ToggleSpectatorPayload,
  UpdateNamePayload,
  UpdateStoryPayload,
  VotePayload,
  WebSocketMessage
} from '@/components/messages/websocket';
import { useRoom } from '@/context/room/roomContext';
import { useToast } from '@/context/toast/toastContext';
import { Eye, EyeOff, Repeat, RotateCcw, Shield, Users, X } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import Header from './page.header';
import LoadingSpinner from '@/components/loadingSpinner/loadingSpinner';
import ParticipantIdBadge from '@/components/participantIdBadge/participantIdBadge';
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
  const [isEditingStory, setIsEditingStory] = useState(false);
  const [isConnected, setIsConnected] = useState(true);
  const [isAuthorized, setIsAuthorized] = useState(false);
  const deliberateDisconnect = useRef(false);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);

  // Planning poker cards (Fibonacci sequence + special cards)
  const cards = ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'];

  useEffect(() => {
    if (isValidRoomId(routeRoomId)) {
      setRoomId(routeRoomId);
      return;
    }

    setRoomId('');
    pushError('Invalid room code. Redirecting to join page.');
    router.replace('/join');
  }, [routeRoomId, router, pushError]);

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
  }, [roomId, router]);

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
          setParticipants(data.participants);
          setCurrentStory(data.currentStory);
          setIsRevealed(data.reveal);
          setSelectedCard(getCurrentUser()?.vote ?? null);
          setResult(data.result ?? null);
          setMostAppearingVotes(data.mostAppearingVotes ?? []);

        } else if (data.type === 'update-client-id') {
          setClientId(data.clientId);
          sessionStorage.setItem('clientId', data.clientId);
          const payload: UpdateNamePayload = { username: userName };
          sendMessage<UpdateNamePayload>({ type: 'update-name', payload });

        } else if (data.type === 'kicked') {
          deliberateDisconnect.current = true;
          cancelReconnect();
          sessionStorage.removeItem('clientId');
          pushError('You have been kicked from the room');
          router.push('/');

        } else {
          throw new Error('Invalid message from websocket');
        }
      } catch (err: any) {
        const message = err?.message ? `Error while handling websocket message: ${err.message}` : 'Error while handling websocket message';
        pushError(message);
      }
    };

    ws.onopen = () => {
      setIsConnected(true);
      reconnectAttemptsRef.current = 0;
      pushSuccess('Connected');
    };

    ws.onclose = () => {
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

    ws.onerror = () => {
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
    router.push('/');
  };

  const votedCount = participants.filter(p => !p.isSpectator && p.hasVoted).length;
  const totalVoters = participants.filter(p => !p.isSpectator).length;

  const amIAdmin = isAdmin();

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
              <h2 style={styles.storyTitle}>Current Story</h2>
              {amIAdmin ? (
                <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', marginTop: '0.5rem' }}>
                  {isEditingStory ? (
                    <>
                      <FocusableComponent
                        currentStory={currentStory}
                        onChange={e => setCurrentStory(e.target.value)}
                        onKeyDown={e => {
                          if (e.key === 'Enter') {
                            const payload: UpdateStoryPayload = { story: currentStory };
                            sendMessage<UpdateStoryPayload>({ type: 'update-story', payload });
                            setIsEditingStory(false);
                          }
                        }}
                      />
                      <button
                        style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                        onClick={() => {
                          const payload: UpdateStoryPayload = { story: currentStory };
                          sendMessage<UpdateStoryPayload>({ type: 'update-story', payload });
                          setIsEditingStory(false);
                        }}
                      >
                        Save
                      </button>
                    </>
                  ) : (
                    <>
                      <label style={{ ...styles.label, margin: 0, flex: 1 }}>{currentStory}</label>
                      <button
                        style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                        onClick={() => setIsEditingStory(!isEditingStory)}
                      >
                        Edit
                      </button>
                    </>
                  )}
                </div>
              ) : (
                <p style={styles.storyText}>{currentStory}</p>
              )}
            </div>
          </div>

          <div className={gridStyles.grid}>
            {/* Main Voting Area */}
            <div style={styles.card}>
              {/* User Info */}
              <div style={styles.userInfo}>
                <div style={styles.inputGroup}>
                  {/* <label style={styles.label}>Your Name</label> */}
                  <label style={styles.label}>{userName}</label>
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
                          : styles.participantWaiting)
                    }}
                  >
                    <div style={styles.participantContent}>
                      <div>
                        <div style={styles.participantName}>
                          {participant.name}
                          {amIAdmin && (
                            <ParticipantIdBadge
                              participantId={participant.id}
                              onCopied={() => pushSuccess('Participant ID copied!')}
                            />
                          )}
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
                    <div>Average: {result?.toFixed(1)}</div>
                    <div>Most Common: {mostAppearingVotes.join(", ")}</div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </Header>
  );
}
