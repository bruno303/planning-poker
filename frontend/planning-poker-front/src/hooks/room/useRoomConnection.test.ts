import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useRoomConnection } from './useRoomConnection';

const socketState: { current: FakeSocket | null } = { current: null };
const connected = { current: false };
const pushError = vi.fn();
const pushSuccess = vi.fn();
const logger = { info: vi.fn(), warn: vi.fn(), error: vi.fn(), setContext: vi.fn() };
const router = { push: vi.fn() };

const deck = { id: 'fibonacci', name: 'Fibonacci', kind: 'numeric', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] };

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
      type: 'room-state', deck, currentStory: 'Story', reveal: false, mostAppearingVotes: [], roomVersion: 7,
      participants: [],
    }) }));

    act(() => result.current.sendMessage({ type: 'reveal-votes', payload: null }));
    expect(JSON.parse(socket.sent[0])).toEqual({ type: 'update-name', payload: { username: 'Ada' } });
    expect(JSON.parse(socket.sent[1])).toEqual({ type: 'reveal-votes', payload: { expectedRoomVersion: 7 } });
  });

  it('normalizes a T-shirt deck descriptor and label summaries into the snapshot', () => {
    const { result } = renderHook(() => useRoomConnection({ roomId: 'room', userName: 'Ada', enabled: true }));
    const socket = socketState.current!;
    const tShirtDeck = { id: 't-shirt', name: 'T-shirt', kind: 'categorical', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] };
    act(() => socket.onmessage?.({ data: JSON.stringify({
      type: 'room-state', deck: tShirtDeck, currentStory: 'Story', reveal: true,
      mostAppearingVotes: ['M'], lowestVote: 'S', highestVote: 'XL', voteSpread: 3, specialVoteCount: 2,
      roomVersion: 9, participants: [],
    }) }));

    expect(result.current.snapshot.deck).toEqual(tShirtDeck);
    expect(result.current.snapshot.mostAppearingVotes).toEqual(['M']);
    expect(result.current.snapshot.lowestVote).toBe('S');
    expect(result.current.snapshot.highestVote).toBe('XL');
    expect(result.current.snapshot.voteSpread).toBe(3);
    expect(result.current.snapshot.specialVoteCount).toBe(2);
    expect(result.current.snapshot.result).toBeNull();
  });

  it('ignores callbacks from a replaced socket and cleans up on unmount', () => {
    const { result, rerender, unmount } = renderHook(({ roomId }) => useRoomConnection({ roomId, userName: 'Ada', enabled: true }), {
      initialProps: { roomId: 'one' },
    });
    const firstSocket = socketState.current!;
    act(() => firstSocket.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'old' }) }));
    act(() => firstSocket.onmessage?.({ data: JSON.stringify({
      type: 'room-state', deck, currentStory: 'Old story', reveal: false, mostAppearingVotes: [], roomVersion: 7,
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
