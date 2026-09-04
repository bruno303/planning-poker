'use client';

import { isRFC3339Timestamp } from '@/components/messages/websocket';
import { useEffect, useState, type CSSProperties } from 'react';

export function formatElapsedTime(totalSeconds: number): string {
  const elapsedSeconds = Math.max(0, Math.floor(totalSeconds));
  const seconds = elapsedSeconds % 60;
  const totalMinutes = Math.floor(elapsedSeconds / 60);
  const minutes = totalMinutes % 60;
  const hours = Math.floor(totalMinutes / 60);

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
  }

  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

function getElapsedSeconds(startedAt: string | null, now = Date.now()): number {
  if (!startedAt || !isRFC3339Timestamp(startedAt)) {
    return 0;
  }

  return Math.max(0, Math.floor((now - Date.parse(startedAt)) / 1000));
}

export function formatVotedAt(votedAt: string | undefined, startedAt: string | null): string | null {
  if (!startedAt || !isRFC3339Timestamp(startedAt) || !isRFC3339Timestamp(votedAt)) {
    return null;
  }

  return formatElapsedTime(getElapsedSeconds(startedAt, Date.parse(votedAt)));
}

type RoomClockProps = {
  startedAt: string | null
  style?: CSSProperties
}

export function RoomClock({ startedAt, style }: RoomClockProps) {
  const [elapsedSeconds, setElapsedSeconds] = useState(() => getElapsedSeconds(startedAt));

  useEffect(() => {
    const updateElapsedTime = () => {
      setElapsedSeconds(getElapsedSeconds(startedAt));
    };

    updateElapsedTime();
    if (!startedAt || !isRFC3339Timestamp(startedAt)) {
      return;
    }

    const interval = window.setInterval(updateElapsedTime, 1000);
    return () => window.clearInterval(interval);
  }, [startedAt]);

  return (
    <time role="timer" aria-label="Elapsed room time" style={style}>
      {formatElapsedTime(elapsedSeconds)}
    </time>
  );
}
