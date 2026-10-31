'use client';

import {
  UpdateNamePayload,
  type ClientCommand,
  isGuardedCommand,
  isServerEvent,
  SERVER_EVENT_TYPES,
} from '@/components/messages/websocket';
import { useLogger } from '@/context/logger/loggerContext';
import { useRoom } from '@/context/room/roomContext';
import { useToast } from '@/context/toast/toastContext';
import { normalizeRoomState, EMPTY_DECK, type RoomSnapshot } from '@/hooks/room/roomState';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';

const RECONNECT_INITIAL_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;
const RECONNECT_MULTIPLIER = 2;
const SERVER_EVENT_TYPE_SET: ReadonlySet<string> = new Set(SERVER_EVENT_TYPES);

const emptySnapshot: RoomSnapshot = {
  deck: EMPTY_DECK, currentStory: '', reveal: false, result: null, mostAppearingVotes: [], consensus: null,
  lowestVote: null, highestVote: null, voteRange: null, voteSpread: null, specialVoteCount: 0,
  participants: [], startedAt: null, backlogMode: false, stories: [], currentStoryIndex: 0, roomVersion: null,
};

declare global {
  interface Window {
    __ws?: WebSocket;
    planning_poker?: { clientID?: string };
  }
}

function getErrorMessage(error: unknown): string | undefined {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  if (error !== null && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message;
  }
  return undefined;
}

export type UseRoomConnectionOptions = {
  roomId: string;
  userName: string;
  enabled: boolean;
};

export function useRoomConnection({ roomId, userName, enabled }: UseRoomConnectionOptions) {
  const logger = useLogger('room-page');
  const router = useRouter();
  const { socket, connected } = useRoom();
  const { pushError, pushSuccess } = useToast();
  const [snapshot, setSnapshot] = useState<RoomSnapshot>(emptySnapshot);
  const [clientId, setClientId] = useState('');
  const [isConnected, setIsConnected] = useState(true);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const deliberateDisconnectRef = useRef(false);
  const clientIdRef = useRef(clientId);
  const roomVersionRef = useRef(snapshot.roomVersion);
  const roomIdRef = useRef(roomId);
  const userNameRef = useRef(userName);
  const sendMessageRef = useRef<(message: ClientCommand) => void>(() => undefined);

  clientIdRef.current = clientId;
  roomVersionRef.current = snapshot.roomVersion;
  roomIdRef.current = roomId;
  userNameRef.current = userName;

  const cancelReconnect = useCallback(() => {
    if (reconnectTimeoutRef.current !== null) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  const disconnect = useCallback(() => {
    deliberateDisconnectRef.current = true;
    cancelReconnect();
    clientIdRef.current = '';
    roomVersionRef.current = null;
    setClientId('');
    setSnapshot(emptySnapshot);
    if (window.planning_poker) window.planning_poker.clientID = undefined;
    const activeSocket = socket.current;
    if (activeSocket) {
      activeSocket.onopen = null;
      activeSocket.onmessage = null;
      activeSocket.onclose = null;
      activeSocket.onerror = null;
      activeSocket.close();
    }
    connected.current = false;
    socket.current = null;
    setIsConnected(false);
  }, [cancelReconnect, connected, socket]);

  const sendMessage = useCallback((message: ClientCommand) => {
    const activeSocket = socket.current;
    if (activeSocket?.readyState !== WebSocket.OPEN) {
      pushError('Connection is not ready. Please wait and try again.');
      return;
    }
    const guarded = isGuardedCommand(message.type);
    if (guarded && roomVersionRef.current === null) {
      pushError('Room state is still loading. Please wait and try again.');
      return;
    }
    try {
      const payload = guarded
        ? { ...(message.payload as unknown as Record<string, unknown>), expectedRoomVersion: roomVersionRef.current }
        : message.payload;
      activeSocket.send(JSON.stringify({ ...message, payload }));
    } catch (error: unknown) {
      const detail = getErrorMessage(error);
      pushError(detail ? `Failed to send message: ${detail}` : 'Failed to send message.');
    }
  }, [pushError, socket]);

  sendMessageRef.current = sendMessage;

  useEffect(() => {
    if (!enabled || !roomId || !userName) return;
    let disposed = false;

    const connect = () => {
      if (disposed) return;
      deliberateDisconnectRef.current = false;
      const savedClientId = sessionStorage.getItem('clientId');
      const wsUrl = savedClientId
        ? `${process.env.NEXT_PUBLIC_WEBSOCKET_URL}/planning/${roomId}/ws?clientId=${encodeURIComponent(savedClientId)}`
        : `${process.env.NEXT_PUBLIC_WEBSOCKET_URL}/planning/${roomId}/ws`;
      const ws = new WebSocket(wsUrl);
      socket.current = ws;
      connected.current = true;
      if (process.env.NODE_ENV !== 'production' || process.env.NEXT_PUBLIC_EXPOSE_WS_GLOBAL === 'true') window.__ws = ws;

      ws.onmessage = (event) => {
        if (disposed || socket.current !== ws) return;
        try {
          const data: unknown = JSON.parse(event.data);
          const frameType = typeof data === 'object' && data !== null && 'type' in data ? data.type : undefined;
          if (typeof frameType === 'string' && !SERVER_EVENT_TYPE_SET.has(frameType)) {
            logger.warn('Ignoring unknown websocket frame', { type: frameType });
            return;
          }
          if (!isServerEvent(data)) {
            throw new Error('Invalid message from websocket');
          }
          switch (data.type) {
            case 'room-state': {
              const nextSnapshot = normalizeRoomState(data);
              roomVersionRef.current = nextSnapshot.roomVersion;
              setSnapshot(nextSnapshot);
              break;
            }
            case 'stale-command': {
              pushError('Room changed; review and try again.');
              break;
            }
            case 'update-client-id': {
              clientIdRef.current = data.clientId;
              setClientId(data.clientId);
              window.planning_poker = { clientID: data.clientId };
              sessionStorage.setItem('clientId', data.clientId);
              logger.setContext({ clientId: data.clientId });
              const payload: UpdateNamePayload = { username: userNameRef.current };
              sendMessageRef.current({ type: 'update-name', payload });
              break;
            }
            case 'kicked': {
              logger.info('Kicked from room');
              disconnect();
              sessionStorage.removeItem('clientId');
              logger.setContext({ clientId: undefined, roomId: undefined });
              pushError('You have been kicked from the room');
              router.push('/');
              break;
            }
            case 'command-ack': {
              logger.info('Command acknowledged', { commandId: data.commandId, roomVersion: data.roomVersion });
              break;
            }
            case 'command-error': {
              logger.info('Command failed', { commandId: data.commandId, code: data.code, message: data.message, roomVersion: data.roomVersion });
              break;
            }
          }
        } catch (error: unknown) {
          const detail = getErrorMessage(error);
          logger.error('Message handling failed', { error: detail });
          pushError(detail ? `Error while handling websocket message: ${detail}` : 'Error while handling websocket message');
        }
      };
      ws.onopen = () => {
        if (disposed || socket.current !== ws) return;
        logger.info('WebSocket connected');
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
        pushSuccess('Connected');
      };
      ws.onclose = (event) => {
        if (disposed || socket.current !== ws) return;
        logger.warn('WebSocket closed', { code: event.code, reason: event.reason });
        setIsConnected(false);
        socket.current = null;
        connected.current = false;
        if (!deliberateDisconnectRef.current) {
          const delay = Math.min(RECONNECT_INITIAL_DELAY * Math.pow(RECONNECT_MULTIPLIER, reconnectAttemptsRef.current), RECONNECT_MAX_DELAY);
          reconnectAttemptsRef.current++;
          logger.warn('Reconnection attempt', { attempt: reconnectAttemptsRef.current });
          reconnectTimeoutRef.current = setTimeout(connect, delay);
        }
      };
      ws.onerror = (event) => {
        if (disposed || socket.current !== ws) return;
        logger.error('WebSocket error', { error: event.type });
        pushError('Connection error');
        connected.current = false;
      };
    };

    connect();
    return () => {
      disposed = true;
      disconnect();
    };
  }, [cancelReconnect, connected, disconnect, enabled, logger, pushError, pushSuccess, roomId, router, socket]);

  return { snapshot, clientId, isConnected, sendMessage, disconnect };
}
