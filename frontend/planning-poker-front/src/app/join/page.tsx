'use client'

import { useLogger } from '@/context/logger/loggerContext';
import { useToast } from '@/context/toast/toastContext';
import { FALLBACK_DECK, fetchDeckPresets, type DeckPreset } from '@/lib/decks';
import { Loader2, LogIn, Plus } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import React, { useEffect, useState } from 'react';
import { styles } from './page.styles';

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    return error.message.trim() ? error.message : fallback;
  }
  if (typeof error === 'string') {
    return error.trim() ? error : fallback;
  }
  if (error !== null && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message.trim() ? error.message : fallback;
  }
  return fallback;
}

export default function PlanningPokerHome() {
  const router = useRouter();
  const params = useParams<{ roomId?: string }>();
  const [roomCode, setRoomCode] = useState('');
  const [userName, setUserName] = useState('');
  const [deckPresets, setDeckPresets] = useState<DeckPreset[]>([FALLBACK_DECK]);
  const [selectedDeckId, setSelectedDeckId] = useState(FALLBACK_DECK.id);
  const [isCreating, setIsCreating] = useState(false);
  const [isJoining, setIsJoining] = useState(false);
  const nameInputRef = React.useRef<HTMLInputElement | null>(null);
  const routeRoomId = typeof params?.roomId === 'string' ? params.roomId : '';
  const hasRoomParam = Boolean(routeRoomId);
  const logger = useLogger('join-page');
  const { pushError } = useToast();

  const getRoomRoute = (value: string) => `/room/${encodeURIComponent(value.trim())}`;

  useEffect(() => {
    setRoomCode(routeRoomId);
  }, [routeRoomId]);

  useEffect(() => { nameInputRef.current?.focus(); }, []);

  useEffect(() => {
    if (hasRoomParam) return;
    let cancelled = false;

    fetchDeckPresets().then((decks) => {
      if (cancelled) return;
      setDeckPresets(decks);
      setSelectedDeckId(decks[0]?.id ?? FALLBACK_DECK.id);
      if (decks.length === 1 && decks[0].id === FALLBACK_DECK.id) {
        logger.warn('Failed to load voting decks; using the fallback deck');
      }
    });

    return () => { cancelled = true; };
  }, [hasRoomParam, logger]);

  const handleCreateRoom = async () => {
    if (!userName.trim()) {
      logger.warn('Validation failed', { reason: 'Name not informed' });
      pushError('Name not informed');
      return;
    }

    try {
      setIsCreating(true);
      const res = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/planning/rooms`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ deckPreset: selectedDeckId }),
      });
      if (!res.ok) {
        throw new Error('Failed to create room on server');
      }
      const data = await res.json();
      logger.info('Room created', { roomId: data.roomId });
      logger.setContext({ roomId: data.roomId });
      sessionStorage.setItem('userName', userName.trim());
      router.push(getRoomRoute(data.roomId));
    } catch (err: unknown) {
      const message = getErrorMessage(err, 'Failed to create room. Please try again.');
      logger.error('Failed to create room', { error: message });
      pushError(message);
    } finally {
      setIsCreating(false);
    }
  };

  const handleJoinRoom = async () => {
    if (!userName.trim()) {
      logger.warn('Validation failed', { reason: 'Name not informed' });
      pushError('Name not informed');
      return;
    }

    if (!roomCode.trim()) {
      logger.warn('Validation failed', { reason: 'Room code not informed' });
      pushError('Room code not informed');
      return;
    }

    try {
      setIsJoining(true);
      sessionStorage.setItem('userName', userName.trim());
      logger.setContext({ roomId: roomCode.trim() });
      router.push(getRoomRoute(roomCode));
    } catch (err: unknown) {
      const message = getErrorMessage(err, 'Failed to join room. Please try again.');
      logger.error('Failed to join room', { error: message });
      pushError(message);
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
            <label htmlFor="user-name" style={styles.label}>Your Name</label>
            <input
              ref={nameInputRef}
              id="user-name"
              type="text"
              autoComplete="name"
              value={userName}
              onChange={(e) => setUserName(e.target.value)}
              placeholder="Enter your name"
              style={styles.input}
              onFocus={(e) => e.target.style.borderColor = '#3b82f6'}
              onBlur={(e) => e.target.style.borderColor = '#e5e7eb'}
              onKeyDown={async (e) => await handleEnterPressed(e)}
            />
          </div>
          {!hasRoomParam && (
            <div style={styles.inputGroup}>
              <label htmlFor="deck-preset" style={styles.label}>Voting deck</label>
              <select
                id="deck-preset"
                value={selectedDeckId}
                onChange={(e) => setSelectedDeckId(e.target.value)}
                style={styles.select}
              >
                {deckPresets.map((deck) => <option key={deck.id} value={deck.id}>{deck.name}</option>)}
              </select>
            </div>
          )}
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
