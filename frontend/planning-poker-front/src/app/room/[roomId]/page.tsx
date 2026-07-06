'use client'


import FocusableComponent from '@/components/focusableInput/focusableInput';
import LoadingSpinner from '@/components/loadingSpinner/loadingSpinner';
import {
  AddStoryPayload,
  RemoveStoryPayload,
  Story,
  ToggleOwnerPayload,
  ToggleSpectatorPayload,
  UpdateNamePayload,
  UpdateStoryPayload,
  VotePayload,
  WebSocketMessage
} from '@/components/messages/websocket';
import ParticipantIdBadge from '@/components/participantIdBadge/participantIdBadge';
import { useRoom } from '@/context/room/roomContext';
import { useToast } from '@/context/toast/toastContext';
import { Eye, EyeOff, List, Plus, Repeat, RotateCcw, Shield, Trash2, Users, X } from 'lucide-react';
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
  const [backlogMode, setBacklogMode] = useState(false);
  const [stories, setStories] = useState<Story[]>([]);
  const [currentStoryIndex, setCurrentStoryIndex] = useState(0);
  const [newStoryInput, setNewStoryInput] = useState('');
  const [showDisableBacklogConfirm, setShowDisableBacklogConfirm] = useState(false);
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

  const handleToggleBacklogMode = () => {
    const payload: any = null;
    sendMessage<any>({ type: 'toggle-backlog-mode', payload });
  };

  const handleAddStory = () => {
    if (!newStoryInput.trim()) return;
    const payload: AddStoryPayload = { story: newStoryInput.trim() };
    sendMessage<AddStoryPayload>({ type: 'add-story', payload });
    setNewStoryInput('');
  };

  const handleRemoveStory = (index: number) => {
    const payload: RemoveStoryPayload = { storyIndex: index };
    sendMessage<RemoveStoryPayload>({ type: 'remove-story', payload });
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
          setBacklogMode(data.backlogMode ?? false);
          setStories(data.stories ?? []);
          setCurrentStoryIndex(data.currentStoryIndex ?? 0);

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
                      <label style={{ ...styles.label, margin: 0, flex: 1 }}>
                        {currentStory}
                        {backlogMode && stories.length > 0 && (
                          <span style={styles.backlogStoryPosition}>
                            (Story {currentStoryIndex + 1} of {stories.length})
                          </span>
                        )}
                      </label>
                      {((backlogMode && currentStory) || !backlogMode) && (
                        <button
                          style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                          onClick={() => setIsEditingStory(!isEditingStory)}
                        >
                          Edit
                        </button>
                      )}
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
                  {backlogMode && currentStoryIndex > 0 && (
                    <button
                      onClick={handlePrevStory}
                      style={{ ...styles.button, ...styles.primaryButton }}
                      onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#2563eb'}
                      onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}
                    >
                      <List size={20} />
                      Previous Story
                    </button>
                  )}
                  {backlogMode && stories.length > 0 && currentStoryIndex < stories.length - 1 && (
                    <button
                      onClick={handleAdvanceStory}
                      style={{ ...styles.button, ...styles.primaryButton }}
                      onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#2563eb'}
                      onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}
                    >
                      <List size={20} />
                      Next Story
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

          {/* Backlog Panel - shown when backlog mode is on */}
          {backlogMode && (
            <div style={styles.backlogPanel}>
              <div style={styles.backlogHeader}>
                <h2 style={styles.sectionTitle}>Story Backlog</h2>
                {amIAdmin && (
                  <button
                    style={{ ...styles.button, ...styles.dangerButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                    onClick={() => setShowDisableBacklogConfirm(true)}
                  >
                    Disable Backlog
                  </button>
                )}
              </div>

              {/* Story list */}
              {stories.length > 0 && (
                <div style={styles.backlogList}>
                  {stories.map((story, index) => (
                    <div
                      key={index}
                      style={{
                        ...styles.backlogStory,
                        ...(index === currentStoryIndex ? styles.backlogStoryCurrent : {}),
                        ...(story.voted ? styles.backlogStoryVoted : index < currentStoryIndex ? styles.backlogStoryVoted : styles.backlogStoryPending),
                      }}
                    >
                      <div style={styles.backlogStoryLeft}>
                        <span style={styles.backlogStoryIndex}>{index + 1}.</span>
                        <span style={{
                          ...styles.backlogStoryName,
                          ...(story.voted ? { textDecoration: 'line-through', opacity: 0.7 } : {}),
                        }}>
                          {story.name}
                        </span>
                        {/* Status indicator */}
                        {index === currentStoryIndex && !story.voted && (
                          <span style={styles.backlogStoryTag}>Current</span>
                        )}
                        {story.voted && story.result != null && (
                          <span style={styles.backlogStoryTagVoted}>
                            Avg: {story.result.toFixed(1)}
                          </span>
                        )}
                      </div>
                      <div style={styles.backlogStoryRight}>
                        {/* Remove button - only for pending stories */}
                        {amIAdmin && !story.voted && index !== currentStoryIndex && (
                          <button
                            style={{ ...styles.button, ...styles.dangerSmallButton }}
                            onClick={() => handleRemoveStory(index)}
                            title="Remove story"
                          >
                            <Trash2 size={14} />
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Add story input - admin only */}
              {amIAdmin && (
                <div style={styles.backlogAddForm}>
                  <input
                    type="text"
                    value={newStoryInput}
                    onChange={e => setNewStoryInput(e.target.value)}
                    onKeyDown={e => {
                      if (e.key === 'Enter') handleAddStory();
                    }}
                    placeholder="Enter story name..."
                    style={styles.backlogInput}
                  />
                  <button
                    style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                    onClick={handleAddStory}
                  >
                    <Plus size={16} />
                    Add
                  </button>
                </div>
              )}
            </div>
          )}

          {/* Disable Backlog Confirmation Dialog */}
          {showDisableBacklogConfirm && (
            <div style={{
              position: 'fixed',
              top: 0,
              left: 0,
              width: '100vw',
              height: '100vh',
              backgroundColor: 'rgba(0, 0, 0, 0.5)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              zIndex: 1000,
            }}>
              <div style={{
                backgroundColor: 'white',
                borderRadius: '0.5rem',
                padding: '2rem',
                maxWidth: '400px',
                boxShadow: '0 4px 20px rgba(0, 0, 0, 0.15)',
              }}>
                <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.75rem', color: '#1f2937' }}>
                  Disable Backlog Mode
                </h3>
                <p style={{ fontSize: '0.875rem', color: '#6b7280', marginBottom: '1.5rem' }}>
                  This will remove the story backlog and keep only the current story. Are you sure?
                </p>
                <div style={{ display: 'flex', gap: '0.5rem', justifyContent: 'flex-end' }}>
                  <button
                    style={{ ...styles.button, background: '#e5e7eb', color: '#374151' }}
                    onClick={() => setShowDisableBacklogConfirm(false)}
                  >
                    Cancel
                  </button>
                  <button
                    style={{ ...styles.button, ...styles.dangerButton }}
                    onClick={() => {
                      setShowDisableBacklogConfirm(false);
                      handleToggleBacklogMode();
                    }}
                  >
                    Disable
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </Header>
  );
}
