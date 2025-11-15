'use client'

import { Loader2, LogIn, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import React, { useEffect, useState } from 'react';
import { styles } from './page.styles';

export default function PlanningPokerHome({ params }: { params: Promise<{ roomId: string | null }> }) {
  const router = useRouter();
  const [roomCode, setRoomCode] = useState('');
  const [userName, setUserName] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [isJoining, setIsJoining] = useState(false);
  const [error, setError] = useState('');
  const myParams = React.use(params);

  useEffect(() => {
    if (myParams.roomId) {
      setRoomCode(myParams.roomId);
    }
  }, []);

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
      sessionStorage.setItem('userName', userName.trim());
      router.push(`/room/${newRoomId}`);
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

      sessionStorage.setItem('userName', userName.trim());
      router.push(`/room/${roomCode.trim()}`);
    } catch (err: any) {
      setError(err.message || 'Failed to join room. Please try again.');
    } finally {
      setIsJoining(false);
    }
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
