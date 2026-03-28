'use client'

import { useToast } from '@/context/toast/toastContext';
import { Loader2, LogIn, Plus } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import React, { useEffect, useState } from 'react';
import { styles } from './page.styles';

export default function PlanningPokerHome() {
  const router = useRouter();
  const params = useParams<{ roomId?: string }>();
  const [roomCode, setRoomCode] = useState('');
  const [userName, setUserName] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [isJoining, setIsJoining] = useState(false);
  const nameInputRef = React.useRef<HTMLInputElement | null>(null);
  const routeRoomId = typeof params?.roomId === 'string' ? params.roomId : '';
  const hasRoomParam = Boolean(routeRoomId);
  const { pushError } = useToast();

  const getRoomRoute = (value: string) => `/room/${encodeURIComponent(value.trim())}`;

  useEffect(() => {
    setRoomCode(routeRoomId);
  }, [routeRoomId]);

  useEffect(() => { nameInputRef.current?.focus(); }, []);

  const handleCreateRoom = async () => {
    if (!userName.trim()) {
      pushError('Name not informed');
      return;
    }

    try {
      setIsCreating(true);
      const newRoomId = crypto.randomUUID();
      sessionStorage.setItem('userName', userName.trim());
      router.push(getRoomRoute(newRoomId));
    } catch (err: any) {
      const message = err?.message || 'Failed to create room. Please try again.';
      pushError(message);
    } finally {
      setIsCreating(false);
    }
  };

  const handleJoinRoom = async () => {
    if (!userName.trim()) {
      pushError('Name not informed');
      return;
    }

    if (!roomCode.trim()) {
      pushError('Room code not informed');
      return;
    }

    try {
      setIsJoining(true);
      sessionStorage.setItem('userName', userName.trim());
      router.push(getRoomRoute(roomCode));
    } catch (err: any) {
      const message = err?.message || 'Failed to join room. Please try again.';
      pushError(message);
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
              ref={nameInputRef}
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
        </div>

        {/* Buttons */}
        <div style={styles.buttonsContainer}>
          {hasRoomParam ? (
            <button
              onClick={handleJoinRoom}
              disabled={isJoining || isCreating || !userName.trim() || !roomCode.trim()}
              style={{
                ...styles.button,
                ...styles.primaryButton,
                ...(isJoining || isCreating || !userName.trim() || !roomCode.trim() ? styles.buttonDisabled : {})
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
                  Join Room
                </>
              )}
            </button>
          ) : (
            <button
              onClick={handleCreateRoom}
              disabled={isCreating || isJoining || !userName.trim()}
              style={{
                ...styles.button,
                ...styles.primaryButton,
                ...(isCreating || isJoining || !userName.trim() ? styles.buttonDisabled : {})
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
                  Create Room
                </>
              )}
            </button>
          )}
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
