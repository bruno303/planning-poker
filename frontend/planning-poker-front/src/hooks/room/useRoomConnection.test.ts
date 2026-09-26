import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useRoomConnection } from './useRoomConnection';

const socketState: { current: FakeSocket | null } = { current: null };
const connected = { current: false };
const pushError = vi.fn();
const pushSuccess = vi.fn();
const logger = { info: vi.fn(), warn: vi.fn(), error: vi.fn(), setContext: vi.fn() };
const router = { push: vi.fn() };

class FakeSocket {
  static OPEN = 1;
  static CLOSED = 3;
  readyState = FakeSocket.OPEN;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: ((event: { code: number; reason: string }) => void) | null = null;
  onerror: ((event: { type: string }) => void) | null = null;
  constructor(public url: string) { socketState.current = this; }
  send(value: string) { this.sent.push(value); }
  close() { this.readyState = FakeSocket.CLOSED; }
}

vi.mock('next/navigation', () => ({ useRouter: () => router }));
vi.mock('@/context/logger/loggerContext', () => ({ useLogger: () => logger }));
vi.mock('@/context/toast/toastContext', () => ({ useToast: () => ({ pushError, pushSuccess }) }));
vi.mock('@/context/room/roomContext', () => ({ useRoom: () => ({ socket: socketState, connected }) }));

describe('useRoomConnection', () => {
  beforeEach(() => {
    vi.stubGlobal('WebSocket', FakeSocket);
    sessionStorage.clear();
    socketState.current = null;
    connected.current = false;
    pushError.mockReset();
    pushSuccess.mockReset();
    router.push.mockReset();
    delete window.planning_poker;
  });

  it('registers the name and injects the latest room version into guarded commands', () => {
    const { result } = renderHook(() => useRoomConnection({ roomId: 'room', userName: 'Ada', enabled: true }));
    const socket = socketState.current!;
    act(() => socket.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    act(() => socket.onmessage?.({ data: JSON.stringify({
      type: 'room-state', currentStory: 'Story', reveal: false, mostAppearingVotes: [], deck: 't-shirt', deckLabels: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'], roomVersion: 7,
      participants: [],
    }) }));
    expect(result.current.snapshot.deck).toBe('t-shirt');
    expect(result.current.snapshot.deckLabels).toEqual(['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕']);

    act(() => result.current.sendMessage({ type: 'reveal-votes', payload: null }));
    expect(JSON.parse(socket.sent[0])).toEqual({ type: 'update-name', payload: { username: 'Ada' } });
    expect(JSON.parse(socket.sent[1])).toEqual({ type: 'reveal-votes', payload: { expectedRoomVersion: 7 } });
  });

  it('ignores callbacks from a replaced socket and cleans up on unmount', () => {
    const { result, rerender, unmount } = renderHook(({ roomId }) => useRoomConnection({ roomId, userName: 'Ada', enabled: true }), {
      initialProps: { roomId: 'one' },
    });
    const firstSocket = socketState.current!;
    act(() => firstSocket.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'old' }) }));
    act(() => firstSocket.onmessage?.({ data: JSON.stringify({
      type: 'room-state', currentStory: 'Old story', reveal: false, mostAppearingVotes: [], deck: 'fibonacci', deckLabels: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'], roomVersion: 7,
      participants: [],
    }) }));
    act(() => rerender({ roomId: 'two' }));
    const secondSocket = socketState.current!;
    expect(result.current.clientId).toBe('');
    expect(result.current.snapshot.currentStory).toBe('');
    act(() => result.current.sendMessage({ type: 'reveal-votes', payload: null }));
    expect(secondSocket.sent).toHaveLength(0);
    expect(pushError).toHaveBeenCalledWith('Room state is still loading. Please wait and try again.');
    unmount();
    expect(secondSocket.readyState).toBe(FakeSocket.CLOSED);
    expect(secondSocket.onmessage).toBeNull();
  });
});
