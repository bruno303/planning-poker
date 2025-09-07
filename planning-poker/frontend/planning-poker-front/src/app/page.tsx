'use client'

import React, { useEffect, useRef, useState } from 'react';
import { Plus, LogIn, Loader2 } from 'lucide-react';
import PlanningPoker from '@/pages/planningpoker';

export default function PlanningPokerHome({ params }: { params: Promise<{ roomId: string | null }> }) {
  const [currentView, setCurrentView] = useState('home'); // 'home' or 'room'
  const [roomCode, setRoomCode] = useState('');
  const [userName, setUserName] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [isJoining, setIsJoining] = useState(false);
  const [error, setError] = useState('');
  const [currentRoomId, setCurrentRoomId] = useState<string | null>(null);
  const myParams = React.use(params);
  const connected = useRef(false);
  const socket = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (myParams.roomId) {
      setRoomCode(myParams.roomId);
    }
  }, []);

  // Home page view
  const styles = {
    container: {
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '1rem',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    },
    card: {
      backgroundColor: 'white',
      borderRadius: '1rem',
      boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
      padding: '3rem',
      width: '100%',
      maxWidth: '500px'
    },
    header: {
      textAlign: 'center' as const,
      marginBottom: '3rem'
    },
    logo: {
      fontSize: '3rem',
      marginBottom: '1rem'
    },
    title: {
      fontSize: '2rem',
      fontWeight: 'bold',
      color: '#1f2937',
      marginBottom: '0.5rem'
    },
    subtitle: {
      color: '#6b7280',
      fontSize: '1.125rem'
    },
    form: {
      marginBottom: '2rem'
    },
    inputGroup: {
      marginBottom: '1.5rem'
    },
    label: {
      display: 'block',
      fontSize: '0.875rem',
      fontWeight: '500',
      color: '#374151',
      marginBottom: '0.5rem'
    },
    input: {
      width: '100%',
      padding: '0.75rem',
      border: '2px solid #e5e7eb',
      borderRadius: '0.5rem',
      fontSize: '1rem',
      outline: 'none',
      transition: 'border-color 0.2s',
      boxSizing: 'border-box' as const
    },
    inputFocus: {
      borderColor: '#3b82f6'
    },
    error: {
      backgroundColor: '#fef2f2',
      color: '#dc2626',
      padding: '0.75rem',
      borderRadius: '0.5rem',
      marginBottom: '1.5rem',
      fontSize: '0.875rem',
      border: '1px solid #fecaca'
    },
    buttonsContainer: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '1rem'
    },
    button: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      gap: '0.5rem',
      padding: '0.875rem 1.5rem',
      borderRadius: '0.5rem',
      fontWeight: '600',
      fontSize: '1rem',
      transition: 'all 0.2s',
      border: 'none',
      cursor: 'pointer',
      minHeight: '3rem'
    },
    primaryButton: {
      backgroundColor: '#3b82f6',
      color: 'white',
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
    },
    secondaryButton: {
      backgroundColor: '#f3f4f6',
      color: '#374151',
      border: '2px solid #e5e7eb'
    },
    buttonDisabled: {
      opacity: 0.6,
      cursor: 'not-allowed'
    },
    divider: {
      display: 'flex',
      alignItems: 'center',
      margin: '1.5rem 0',
      color: '#9ca3af'
    },
    dividerLine: {
      flex: 1,
      height: '1px',
      backgroundColor: '#e5e7eb'
    },
    dividerText: {
      padding: '0 1rem',
      fontSize: '0.875rem'
    },
    roomHeader: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      padding: '1rem 2rem',
      backgroundColor: 'white',
      boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1)',
      marginBottom: '0'
    },
    backButton: {
      padding: '0.5rem 1rem',
      backgroundColor: '#f3f4f6',
      color: '#374151',
      border: '1px solid #d1d5db',
      borderRadius: '0.375rem',
      cursor: 'pointer',
      fontSize: '0.875rem',
      transition: 'background-color 0.2s'
    },
    roomInfo: {
      display: 'flex',
      alignItems: 'center',
      gap: '0.5rem'
    },
    roomLabel: {
      fontSize: '0.875rem',
      color: '#6b7280'
    },
    roomCode: {
      fontSize: '1rem',
      fontWeight: '600',
      color: '#1f2937',
      backgroundColor: '#f3f4f6',
      padding: '0.25rem 0.5rem',
      borderRadius: '0.25rem',
      fontFamily: 'Monaco, "Lucida Console", monospace'
    },
    placeholder: {
      minHeight: '80vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: '#f9fafb'
    },
    placeholderContent: {
      textAlign: 'center' as const,
      maxWidth: '600px',
      padding: '2rem'
    },
    placeholderTitle: {
      fontSize: '2rem',
      fontWeight: 'bold',
      color: '#1f2937',
      marginBottom: '1rem'
    },
    placeholderText: {
      fontSize: '1.125rem',
      color: '#4b5563',
      marginBottom: '1rem'
    },
    placeholderSubtext: {
      color: '#6b7280',
      fontSize: '1rem',
      fontStyle: 'italic'
    }
  };

  // Mock API call to create a new room
  const createRoom = async (userName: string) => {
    try {
      setIsCreating(true);
      setError('');
      
      // Simulate API call
      const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/planning/room`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          createdBy: userName
        }),
        credentials: 'include'
      });
      
      if (!response.ok) {
        throw new Error('Failed to create room');
      }
      
      const data = await response.json();
      return data.roomId; // Assuming API returns { roomId: "uuid-here" }
      
    } catch (err: any) {
      throw new Error(err.message || 'Failed to create room. Please try again.');
    } finally {
      setIsCreating(false);
    }
  };

  const handleCreateRoom = async () => {
    if (!userName.trim()) {
      setError('Please enter your name');
      return;
    }

    try {
      const newRoomId = await createRoom(userName.trim());
      setRoomCode(newRoomId);
      setCurrentRoomId(newRoomId);
      setCurrentView('room');
    } catch (err: any) {
      setError(err.message || 'Failed to create room. Please try again.');
    }
  };

  const handleJoinRoom = async () => {
    if (!userName.trim()) {
      setError('Please enter your name');
      return;
    }
    
    if (!roomCode.trim()) {
      setError('Please enter a room code');
      return;
    }

    try {

      // Simulate API call to join room
      const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/planning/room/${roomCode}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (response.status === 404) {
        setError('Room not found');
        return
      }

      if (!response.ok) {
        setError('Failed to join room');
        console.log(await response.text());
        return
      }

      setCurrentRoomId(roomCode.trim());
      setCurrentView('room');
    } catch (err: any) {
      setError(err.message || 'Failed to join room. Please try again.');
    }
  };

  const handleBackToHome = () => {
    socket.current?.close();
    connected.current = false;
    setCurrentView('home');
    setCurrentRoomId(null);
    setError('');
  };

  const generateShareableLink = () => {
    // console.log(`${window.location.origin}/room/${currentRoomId}`);
    return `${window.location.origin}/room/${currentRoomId}`;
  };

  const handleEnterPressed = async (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      if (roomCode.trim()) {
        await handleJoinRoom();
      } else {
        await handleCreateRoom();
      }
    }
  };

  // If we're in a room, show the planning poker component
  if (currentView === 'room') {
    return (
      <div>
        {/* Room Header */}
        <div style={styles.roomHeader}>
          <button onClick={handleBackToHome} style={styles.backButton}>
            ← Back to Home
          </button>
          <div style={styles.roomInfo}>
            <span style={styles.roomLabel}>Room Code:</span>
            <span
              style={{
              ...styles.roomCode,
              cursor: 'pointer', // Add pointer cursor
              }}
              onMouseEnter={e => {
              (e.target as HTMLElement).style.backgroundColor = '#e0e7ff';
              (e.target as HTMLElement).style.cursor = 'pointer'; // Ensure pointer on hover
              }}
              onMouseLeave={e => {
              (e.target as HTMLElement).style.backgroundColor = '#f3f4f6';
              }}
              title="Click to copy shareable link"
              onClick={async () => {
              await navigator.clipboard.writeText(generateShareableLink())

              const toast = document.createElement('div');
              toast.textContent = 'Shareable link copied!';
              Object.assign(toast.style, {
                position: 'fixed',
                top: '2rem',
                left: '50%',
                transform: 'translateX(-50%)',
                background: '#3b82f6',
                color: 'white',
                padding: '0.75rem 1.5rem',
                borderRadius: '0.5rem',
                boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
                fontSize: '1rem',
                zIndex: 9999,
                opacity: '0',
                transition: 'opacity 0.3s'
              });
              document.body.appendChild(toast);
              setTimeout(() => {
                toast.style.opacity = '1';
              }, 10);
              setTimeout(() => {
                toast.style.opacity = '0';
                setTimeout(() => document.body.removeChild(toast), 300);
              }, 1800);
              }}
            >{currentRoomId}</span>
          </div>
        </div>

        <PlanningPoker userName={userName} roomId={currentRoomId} socket={socket} connected={connected}></PlanningPoker>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <div style={styles.card}>
        {/* Header */}
        <div style={styles.header}>
          <div style={styles.logo}>🃏</div>
          <h1 style={styles.title}>Planning Poker</h1>
          <p style={styles.subtitle}>Collaborate and estimate together</p>
        </div>

        {/* Form */}
        <div style={styles.form}>
          <div style={styles.inputGroup}>
            <label style={styles.label}>Your Name</label>
            <input
              type="text"
              value={userName}
              onChange={(e) => setUserName(e.target.value)}
              placeholder="Enter your name"
              style={styles.input}
              onFocus={(e) => e.target.style.borderColor = '#3b82f6'}
              onBlur={(e) => e.target.style.borderColor = '#e5e7eb'}
              onKeyDown={async (e) => await handleEnterPressed(e)}
            />
          </div>

          <div style={styles.inputGroup}>
            <label style={styles.label}>Room Code (Optional)</label>
            <input
              type="text"
              value={roomCode}
              onChange={(e) => setRoomCode(e.target.value)}
              placeholder="Enter room code to join existing room"
              style={styles.input}
              onFocus={(e) => e.target.style.borderColor = '#3b82f6'}
              onBlur={(e) => e.target.style.borderColor = '#e5e7eb'}
              onKeyDown={async (e) => await handleEnterPressed(e)}
            />
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div style={styles.error}>
            {error}
          </div>
        )}

        {/* Buttons */}
        <div style={styles.buttonsContainer}>
          {/* Join Room Button */}
          <button
            onClick={handleJoinRoom}
            disabled={isJoining || isCreating || !userName.trim() || !roomCode.trim()}
            style={{
              ...styles.button,
              ...styles.primaryButton,
              ...(isJoining || isCreating || !userName.trim() || !roomCode.trim() ? styles.buttonDisabled : {})
            }}
            onMouseEnter={(e) => {
              if (!isJoining && !isCreating && userName.trim() && roomCode.trim()) {
                (e.target as HTMLElement).style.backgroundColor = '#2563eb';
              }
            }}
            onMouseLeave={(e) => {
              if (!isJoining && !isCreating) {
                (e.target as HTMLElement).style.backgroundColor = '#3b82f6';
              }
            }}
          >
            {isJoining ? (
              <>
                <Loader2 size={20} style={{ animation: 'spin 1s linear infinite' }} />
                Joining Room...
              </>
            ) : (
              <>
                <LogIn size={20} />
                Join Existing Room
              </>
            )}
          </button>

          {/* Divider */}
          <div style={styles.divider}>
            <div style={styles.dividerLine}></div>
            <span style={styles.dividerText}>or</span>
            <div style={styles.dividerLine}></div>
          </div>

          {/* Create Room Button */}
          <button
            onClick={handleCreateRoom}
            disabled={isCreating || isJoining || !userName.trim()}
            style={{
              ...styles.button,
              ...styles.secondaryButton,
              ...(isCreating || isJoining || !userName.trim() ? styles.buttonDisabled : {})
            }}
            onMouseEnter={(e) => {
              if (!isCreating && !isJoining && userName.trim()) {
                (e.target as HTMLElement).style.backgroundColor = '#e5e7eb';
              }
            }}
            onMouseLeave={(e) => {
              if (!isCreating && !isJoining) {
                (e.target as HTMLElement).style.backgroundColor = '#f3f4f6';
              }
            }}
          >
            {isCreating ? (
              <>
                <Loader2 size={20} style={{ animation: 'spin 1s linear infinite' }} />
                Creating Room...
              </>
            ) : (
              <>
                <Plus size={20} />
                Create New Room
              </>
            )}
          </button>
        </div>

        <style>
          {`
            @keyframes spin {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
          `}
        </style>
      </div>
    </div>
  );
}
