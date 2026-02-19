'use client'

import FocusableComponent from '@/components/focusableInput/focusableInput';
import { useRoom } from '@/context/room/roomContext';
import { useToast } from '@/context/toast/toastContext';
import { Eye, EyeOff, Repeat, RotateCcw, Shield, Users, X } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import Header from './page.header';
import gridStyles from './page.module.css';
import { styles } from './page.styles';

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
  const roomId = params?.roomId as string ?? '00000000-0000-0000-0000-000000000000';

  const [selectedCard, setSelectedCard] = useState<Card>(null);
  const [userName, setUserName] = useState('');
  const [isRevealed, setIsRevealed] = useState(false);
  const [currentStory, setCurrentStory] = useState('');
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [clientId, setClientId] = useState('');
  const [result, setResult] = useState<number | null>(null);
  const [mostAppearingVotes, setMostAppearingVotes] = useState<number[]>([]);
  const [isEditingStory, setIsEditingStory] = useState(false);

  // Planning poker cards (Fibonacci sequence + special cards)
  const cards = ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'];

  useEffect(() => {
    const storedUserName = sessionStorage.getItem('userName');
    if (!storedUserName) {
      router.push(`/join/${roomId}`);
      return;
    }
    setUserName(storedUserName);

    if (!connected.current) {
      connectWebSocket(roomId, storedUserName);
      connected.current = true;
    }
  }, [roomId]);

  useEffect(() => {
    if (clientId && participants.length > 0) {
      setSelectedCard(getCurrentUser()?.vote ?? null);
    }
  }, [participants, clientId]);

  const handleCardSelect = (card: Card) => {
    if (!isRevealed) {
      socket.current?.send(JSON.stringify({ roomId: roomId, type: 'vote', vote: card, clientId: clientId }));
    }
  };

  const handleRevealVotes = () => {
    socket.current?.send(JSON.stringify({ roomId: roomId, type: 'reveal-votes', clientId: clientId }));
  };

  const handleNewVoting = () => {
    socket.current?.send(JSON.stringify({ roomId: roomId, type: 'new-voting', clientId: clientId }));
  };

  const handleToggleSpectator = (participantId: string) => {
    socket.current?.send(JSON.stringify({
      roomId: roomId,
      type: 'toggle-spectator',
      targetClientId: participantId,
      clientId: clientId
    }));
  };

  const handleToggleAdmin = (participantId: string) => {
    socket.current?.send(JSON.stringify({
      roomId: roomId,
      type: 'toggle-owner',
      targetClientId: participantId,
      clientId: clientId
    }));
  };

  const handleVoteAgain = () => {
    socket.current?.send(JSON.stringify({ roomId: roomId, type: 'vote-again', clientId: clientId }));
  }

  const connectWebSocket = async (roomCode: string | null, userName: string) => {
    if (!roomCode) {
      return
    }
    socket.current = new WebSocket(`${process.env.NEXT_PUBLIC_WEBSOCKET_URL}/planning/${roomCode}/ws`);

    socket.current.onmessage = (event) => {
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
          const updateClientMessage = { roomId: roomId, type: 'update-name', username: userName, clientId: data.clientId }
          socket.current?.send(JSON.stringify(updateClientMessage));

        } else {
          throw new Error('Invalid message from websocket');
        }
      } catch (err: any) {
        const message = err?.message ? `Error while handling websocket message: ${err.message}` : 'Error while handling websocket message';
        pushError(message);
      }
    };

    socket.current.onopen = () => {
      pushSuccess('Connected');
    };

    socket.current.onclose = () => {
      pushError('Disconnected');
    };

    socket.current.onerror = () => {
      pushError('Error occurred while connecting to websocket');
    };
  }

  const getCurrentUser = () => {
    return participants.filter((p: any) => p.id == clientId)[0];
  }

  const isAdmin = (): Boolean => {
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
    socket.current?.close();
    connected.current = false;
    router.push('/');
  };

  const votedCount = participants.filter(p => !p.isSpectator && p.hasVoted).length;
  const totalVoters = participants.filter(p => !p.isSpectator).length;

  return (

    <Header
      handleBackToHome={handleBackToHome}
      generateShareableLink={() => `${window.location.origin}/room/${roomId}`}
    >
      <div style={styles.container}>
        <div style={styles.maxWidth}>
          {/* Header */}
          <div style={styles.header}>
            <h1 style={styles.title}>Planning Poker</h1>

            <div style={styles.storyCard}>
              <h2 style={styles.storyTitle}>Current Story</h2>
              {isAdmin() ? (
                <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', marginTop: '0.5rem' }}>
                  {isEditingStory ? (
                    <>
                      <FocusableComponent
                        currentStory={currentStory}
                        onChange={e => setCurrentStory(e.target.value)}
                        onKeyDown={e => {
                          if (e.key === 'Enter') {
                            socket.current?.send(JSON.stringify({ roomId: roomId, clientId: clientId, type: 'update-story', story: currentStory }));
                            setIsEditingStory(false);
                          }
                        }}
                      />
                      <button
                        style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                        onClick={() => {
                          socket.current?.send(JSON.stringify({ roomId: roomId, clientId: clientId, type: 'update-story', story: currentStory }))
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
              {isAdmin() && (
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
                        <div style={styles.participantName}>{participant.name}</div>
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

                        {isAdmin() && (
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
