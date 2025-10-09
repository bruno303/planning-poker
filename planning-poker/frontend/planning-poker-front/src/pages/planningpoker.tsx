'use client'

import { Eye, EyeOff, Repeat, RotateCcw, Shield, Users } from 'lucide-react';
import { RefObject, useEffect, useState } from 'react';

type Card = string | null
type PlanningPokerProps = {
  roomId: string | null
  userName: string
  connected: RefObject<boolean>
  socket: RefObject<WebSocket | null>
}

type Participant = {
  id: string
  name: string
  vote: Card
  hasVoted: boolean
  isSpectator: boolean
  isOwner: boolean
}

export default function PlanningPoker(props: PlanningPokerProps) {
  const [selectedCard, setSelectedCard] = useState<Card>(null);
  const [userName, setUserName] = useState(props.userName);
  const [isRevealed, setIsRevealed] = useState(false);
  const [currentStory, setCurrentStory] = useState('');
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [clientId, setClientId] = useState('');

  // Planning poker cards (Fibonacci sequence + special cards)
  const cards = ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'];

  useEffect(() => {
    if (!props.connected.current) {
      connectWebSocket(props.roomId);
      props.connected.current = true;
    }
  }, []);

  useEffect(() => {
    if (clientId && participants.length > 0) {
      setSelectedCard(getCurrentUser()?.vote ?? null);
    }
  }, [participants, clientId]);

  const handleCardSelect = (card: Card) => {
    if (!isRevealed) {
      props.socket.current?.send(JSON.stringify({ roomId: props.roomId, type: 'vote', vote: card, clientId: clientId }));
    }
  };

  const handleRevealVotes = () => {
    props.socket.current?.send(JSON.stringify({ roomId: props.roomId, type: 'reveal-votes', clientId: clientId }));
  };

  const handleNewVoting = () => {
    props.socket.current?.send(JSON.stringify({ roomId: props.roomId, type: 'new-voting', clientId: clientId }));
  };

  const handleUpdateUsername = (username: string) => {
    setUserName(username);
    props.socket.current?.send(JSON.stringify({ roomId: props.roomId, type: 'update-name', username: username, clientId: clientId }));
  }

  const handleToggleSpectator = (participantId: string) => {
    props.socket.current?.send(JSON.stringify({ 
      roomId: props.roomId,
      type: 'toggle-spectator',
      targetClientId: participantId,
      clientId: clientId
    }));
  };

  const handleToggleAdmin = (participantId: string) => {
    props.socket.current?.send(JSON.stringify({
      roomId: props.roomId,
      type: 'toggle-owner',
      targetClientId: participantId,
      clientId: clientId
    }));
  };

  const handleVoteAgain = () => {
    props.socket.current?.send(JSON.stringify({ roomId: props.roomId, type: 'vote-again', clientId: clientId }));
  }

  const connectWebSocket = async(roomCode: string | null) => {
    if (!roomCode) {
      return
    }

    // Simulate websocket connection to get room data at real time
    props.socket.current = new WebSocket(`${process.env.NEXT_PUBLIC_WEBSOCKET_URL}/planning/${roomCode}/ws`);

    props.socket.current.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        // console.log('Received message from websocket:', data);

        if (data.type === 'room-state') {
          
          setParticipants(data.participants);
          setCurrentStory(data.currentStory);
          setIsRevealed(data.reveal);
          setSelectedCard(getCurrentUser()?.vote ?? null);

        } else if (data.type === 'update-client-id') {
          setClientId(data.clientId);
          props.socket.current?.send(JSON.stringify({ roomId: props.roomId, type: 'update-name', username: props.userName, clientId: data.clientId }));

        } else {
          throw new Error('Invalid message from websocket');
        }
      } catch (err) {
        console.error('Error while handling websocket message:', err);
      }
    };
    
    props.socket.current.onopen = () => {
      console.log('Connected to websocket');
    };
    
    props.socket.current.onclose = () => {
      console.log('Disconnected from websocket');
    };
    
    props.socket.current.onerror = (event) => {
      console.error('Error occurred while connecting to websocket:', event);
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

  const votedCount = participants.filter(p => !p.isSpectator && p.hasVoted).length;
  const totalVoters = participants.filter(p => !p.isSpectator).length;

  const styles = {
    container: {
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #dbeafe 0%, #e0e7ff 100%)',
      padding: '1rem',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    },
    maxWidth: {
      maxWidth: '1280px',
      margin: '0 auto'
    },
    header: {
      textAlign: 'center' as const,
      marginBottom: '2rem'
    },
    title: {
      fontSize: '2.25rem',
      fontWeight: 'bold',
      color: '#1f2937',
      marginBottom: '1rem'
    },
    storyCard: {
      backgroundColor: 'white',
      borderRadius: '0.5rem',
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
      padding: '1rem',
      marginBottom: '1rem'
    },
    storyTitle: {
      fontSize: '1.25rem',
      fontWeight: '600',
      color: '#374151',
      marginBottom: '0.5rem'
    },
    storyText: {
      color: '#6b7280',
      fontStyle: 'italic'
    },
    grid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
      gap: '1.5rem'
    },
    gridLarge: {
      gridColumn: '1 / -1'
    },
    card: {
      backgroundColor: 'white',
      borderRadius: '0.5rem',
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
      padding: '1.5rem'
    },
    userInfo: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'flex-start',
      marginBottom: '1rem',
      flexWrap: 'wrap' as const,
      gap: '1rem'
    },
    inputGroup: {
      display: 'flex',
      flexDirection: 'column' as const
    },
    label: {
      fontSize: '0.875rem',
      fontWeight: '500',
      color: '#374151',
      marginBottom: '0.5rem'
    },
    input: {
      padding: '0.5rem 0.75rem',
      border: '1px solid #d1d5db',
      borderRadius: '0.375rem',
      fontSize: '1rem',
      outline: 'none',
      transition: 'border-color 0.2s'
    },
    voteStats: {
      textAlign: 'right' as const
    },
    voteStatsLabel: {
      fontSize: '0.875rem',
      color: '#6b7280'
    },
    voteStatsNumber: {
      fontSize: '1.5rem',
      fontWeight: 'bold',
      color: '#2563eb'
    },
    selectedCard: {
      textAlign: 'center' as const,
      marginTop: '1rem',
      display: 'flex',
      flexDirection: 'column' as const,
      alignItems: 'center',
      justifyContent: 'center'
    },
    selectedCardLabel: {
      fontSize: '0.875rem',
      color: '#6b7280',
      marginBottom: '0.5rem'
    },
    selectedCardDisplay: {
      width: '4rem',
      height: '5rem',
      borderRadius: '0.5rem',
      color: 'white',
      fontWeight: 'bold',
      fontSize: '1.25rem',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
    },
    sectionTitle: {
      fontSize: '1.125rem',
      fontWeight: '600',
      color: '#374151',
      marginBottom: '1rem'
    },
    cardsGrid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fill, minmax(3rem, 1fr))',
      gap: '0.75rem',
      marginBottom: '1.5rem'
    },
    pokerCard: {
      width: '3rem',
      height: '4rem',
      borderRadius: '0.5rem',
      fontWeight: 'bold',
      color: 'white',
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
      transition: 'all 0.2s',
      border: 'none',
      cursor: 'pointer',
      fontSize: '1rem'
    },
    pokerCardSelected: {
      transform: 'scale(1.05)',
      boxShadow: '0 0 0 4px rgba(59, 130, 246, 0.3)'
    },
    buttonsContainer: {
      display: 'flex',
      gap: '1rem',
      justifyContent: 'center',
      flexWrap: 'wrap' as const
    },
    button: {
      display: 'flex',
      alignItems: 'center',
      gap: '0.5rem',
      padding: '0.75rem 1.5rem',
      borderRadius: '0.5rem',
      fontWeight: '600',
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
      transition: 'background-color 0.2s',
      border: 'none',
      cursor: 'pointer',
      fontSize: '1rem'
    },
    primaryButton: {
      backgroundColor: '#3b82f6',
      color: 'white'
    },
    successButton: {
      backgroundColor: '#10b981',
      color: 'white'
    },
    warningButton: {
      backgroundColor: '#eab308',
      color: 'white'
    },
    participantsHeader: {
      display: 'flex',
      alignItems: 'center',
      gap: '0.5rem',
      marginBottom: '1rem'
    },
    participantsList: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '0.75rem'
    },
    participant: {
      padding: '0.75rem',
      borderRadius: '0.5rem',
      border: '2px solid',
      transition: 'colors 0.2s'
    },
    participantSpectator: {
      borderColor: '#e5e7eb',
      backgroundColor: '#f9fafb'
    },
    participantVoted: {
      borderColor: '#bbf7d0',
      backgroundColor: '#f0fdf4'
    },
    participantWaiting: {
      borderColor: '#fef3c7',
      backgroundColor: '#fffbeb'
    },
    participantContent: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between'
    },
    participantName: {
      fontWeight: '500',
      color: '#1f2937'
    },
    participantStatus: {
      fontSize: '0.875rem',
      color: '#6b7280'
    },
    participantRight: {
      display: 'flex',
      alignItems: 'center',
      gap: '0.5rem'
    },
    voteCard: {
      width: '2rem',
      height: '2.5rem',
      borderRadius: '0.25rem',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      color: 'white',
      fontWeight: 'bold',
      fontSize: '0.875rem'
    },
    statusDot: {
      width: '0.75rem',
      height: '0.75rem',
      borderRadius: '50%'
    },
    statusSpectator: {
      backgroundColor: '#9ca3af'
    },
    statusVoted: {
      backgroundColor: '#10b981'
    },
    statusWaiting: {
      backgroundColor: '#eab308'
    },
    summary: {
      marginTop: '1.5rem',
      padding: '1rem',
      backgroundColor: '#dbeafe',
      borderRadius: '0.5rem'
    },
    summaryTitle: {
      fontWeight: '600',
      color: '#1e40af',
      marginBottom: '0.5rem'
    },
    summaryContent: {
      fontSize: '0.875rem',
      color: '#1d4ed8'
    },
    adminControls: {
      display: 'flex',
      gap: '0.25rem'
    },
    roleButton: {
      width: '1.75rem',
      height: '1.75rem',
      borderRadius: '0.25rem',
      border: '1px solid #d1d5db',
      backgroundColor: 'white',
      cursor: 'pointer',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      transition: 'all 0.2s',
      fontSize: '0.75rem'
    },
    activeSpectatorButton: {
      backgroundColor: '#ddd6fe',
      borderColor: '#8b5cf6',
      color: '#6d28d9'
    },
    activeAdminButton: {
      backgroundColor: '#fef3c7',
      borderColor: '#f59e0b',
      color: '#92400e'
    },
    inactiveButton: {
      backgroundColor: 'white',
      borderColor: '#d1d5db',
      color: '#9ca3af'
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.maxWidth}>
        {/* Header */}
        <div style={styles.header}>
          <h1 style={styles.title}>Planning Poker</h1>
          
          <div style={styles.storyCard}>
            <h2 style={styles.storyTitle}>Current Story</h2>
            {isAdmin() ? (
              <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', marginTop: '0.5rem' }}>
                <input
                  type="text"
                  value={currentStory}
                  onChange={e => setCurrentStory(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter') {
                      props.socket.current?.send(JSON.stringify({ roomId: props.roomId, clientId: clientId, type: 'update-story', story: currentStory }));
                    }
                  }}
                  style={{ ...styles.input, fontStyle: 'italic' }}
                />
                <button
                  style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                  onClick={() => props.socket.current?.send(JSON.stringify({ roomId: props.roomId, clientId: clientId, type: 'update-story', story: currentStory }))}
                >
                  Save
                </button>
              </div>
            ) : (
              <p style={styles.storyText}>{currentStory}</p>
            )}
          </div>
        </div>

        <div style={styles.grid}>
          {/* Main Voting Area */}
          <div style={{...styles.gridLarge, ...styles.card}}>
            {/* User Info */}
            <div style={styles.userInfo}>
              <div style={styles.inputGroup}>
                <label style={styles.label}>Your Name</label>
                <input
                  type="text"
                  value={userName}
                  onChange={(e) => handleUpdateUsername(e.target.value.toUpperCase())}
                  style={styles.input}
                  onFocus={(e) => e.target.style.borderColor = '#3b82f6'}
                  onBlur={(e) => e.target.style.borderColor = '#d1d5db'}
                  disabled={true}
                />
              </div>
              <div style={styles.voteStats}>
                <div style={styles.voteStatsLabel}>Votes Cast</div>
                <div style={styles.voteStatsNumber}>{votedCount}/{totalVoters}</div>
              </div>
            </div>
            
            {selectedCard && (
              <div style={styles.selectedCard}>
                <div style={styles.selectedCardLabel}>Your Vote</div>
                <div style={{...styles.selectedCardDisplay, backgroundColor: getCardColor(selectedCard)}}>
                  {selectedCard}
                </div>
              </div>
            )}

            {/* Planning Poker Cards */}
            <div style={{marginTop: '2rem'}}>
              <h3 style={styles.sectionTitle}>Select Your Card</h3>
              <div style={styles.cardsGrid}>
                {cards.map((card) => (
                  <button
                    key={card}
                    onClick={() => handleCardSelect(card)}
                    style={{
                      ...styles.pokerCard,
                      backgroundColor: getCardColor(card),
                      ...(selectedCard === card ? styles.pokerCardSelected : {})
                    }}
                    onMouseEnter={(e) => {
                      if (selectedCard !== card) {
                        (e.target as HTMLButtonElement).style.transform = 'scale(1.05)';
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (selectedCard !== card) {
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
                  style={{...styles.button, ...styles.primaryButton}}
                  onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#2563eb'}
                  onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}
                >
                  {isRevealed ? <EyeOff size={20} /> : <Eye size={20} />}
                  {isRevealed ? 'Hide Votes' : 'Reveal Votes'}
                </button>
                <button
                  onClick={handleNewVoting}
                  style={{...styles.button, ...styles.successButton}}
                  onMouseEnter={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#059669'}
                  onMouseLeave={(e) => (e.target as HTMLButtonElement).style.backgroundColor = '#10b981'}
                >
                  <RotateCcw size={20} />
                  New Voting
                </button>
                <button
                  onClick={handleVoteAgain}
                  style={{...styles.button, ...styles.warningButton}}
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
                  <div>Average: {(participants
                    .filter(p => !p.isSpectator && p.hasVoted && !isNaN(parseInt(p.vote ?? '')))
                    .reduce((sum, p) => sum + parseInt(p.vote ?? ''), 0) / 
                    participants.filter(p => !p.isSpectator && p.hasVoted && !isNaN(parseInt(p.vote ?? ''))).length
                  ).toFixed(1)}</div>
                  {/* <div>Most Common: 8</div> */}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
