import { act } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Room from './page';

const push = vi.fn();
const replace = vi.fn();
const pushError = vi.fn();
const pushSuccess = vi.fn();
const playNotificationSound = vi.hoisted(() => vi.fn());
const logger = { info: vi.fn(), warn: vi.fn(), error: vi.fn(), setContext: vi.fn() };
const router = { push, replace };
const socketRef: { current: FakeSocket | null } = { current: null };
const connected = { current: false };
let params: { roomId: string } = { roomId: '123e4567-e89b-12d3-a456-426614174000' };

class FakeSocket {
  static OPEN = 1;
  static CLOSED = 3;
  readyState = FakeSocket.OPEN;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: ((event: { code: number; reason: string }) => void) | null = null;
  onerror: ((event: { type: string }) => void) | null = null;
  constructor(public url: string) { socketRef.current = this; }
  send(value: string) { this.sent.push(value); }
  close() { this.readyState = FakeSocket.CLOSED; }
}

vi.mock('next/navigation', () => ({ useParams: () => params, useRouter: () => router }));
vi.mock('@/context/logger/loggerContext', () => ({ useLogger: () => logger }));
vi.mock('@/context/toast/toastContext', () => ({ useToast: () => ({ pushError, pushSuccess }) }));
vi.mock('@/lib/notificationSound', () => ({ playNotificationSound }));
vi.mock('@/context/room/roomContext', () => ({ useRoom: () => ({ socket: socketRef, connected }) }));
vi.mock('@/components/avatar/avatar', () => ({ default: () => <span aria-hidden="true" /> }));
vi.mock('@/components/participantIdBadge/participantIdBadge', () => ({ default: () => null }));
vi.mock('@/components/focusableInput/focusableInput', () => ({ default: (props: { currentStory: string; onChange: React.ChangeEventHandler<HTMLInputElement>; onKeyDown: React.KeyboardEventHandler<HTMLInputElement> }) => <input aria-label="Story editor" value={props.currentStory} onChange={props.onChange} onKeyDown={props.onKeyDown} /> }));

const roomState = (overrides = {}) => ({
  type: 'room-state', currentStory: 'Implement feature', reveal: false, mostAppearingVotes: [], roomVersion: 4,
  participants: [
    { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true },
    { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
  ], ...overrides,
});

function renderRoom() {
  sessionStorage.setItem('userName', 'Ada');
  vi.stubGlobal('WebSocket', FakeSocket);
  return render(<Room />);
}

describe('room page', () => {
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });
  beforeEach(() => {
    params = { roomId: '123e4567-e89b-12d3-a456-426614174000' };
    socketRef.current = null; connected.current = false;
    push.mockReset(); replace.mockReset(); pushError.mockReset(); pushSuccess.mockReset(); playNotificationSound.mockReset();
    sessionStorage.clear(); vi.restoreAllMocks();
    delete window.planning_poker;
  });

  it('redirects invalid routes and routes visitors without a name to join', async () => {
    params = { roomId: 'not-a-uuid' };
    const invalid = renderRoom();
    await waitFor(() => expect(replace).toHaveBeenCalledWith('/join'));
    invalid.unmount();

    params = { roomId: '123e4567-e89b-12d3-a456-426614174000' };
    sessionStorage.clear();
    render(<Room />);
    await waitFor(() => expect(push).toHaveBeenCalledWith('/join/123e4567-e89b-12d3-a456-426614174000'));
  });

  it('connects, renders room state, votes, and performs admin actions', async () => {
    renderRoom();
    await waitFor(() => expect(socketRef.current).not.toBeNull());
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    expect(window.planning_poker?.clientID).toBe('me');
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState()) }));
    await waitFor(() => expect(screen.getByText('Implement feature')).toBeTruthy());
    expect(screen.getAllByText('Ada').length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '8' }));
    fireEvent.click(screen.getByRole('button', { name: /reveal votes/i }));
    expect(ws.sent.map((value) => JSON.parse(value).type)).toEqual(['update-name', 'vote', 'reveal-votes']);
    expect(JSON.parse(ws.sent[1]).payload.vote).toBe('8');
    expect(JSON.parse(ws.sent[2]).payload).toEqual({ expectedRoomVersion: 4 });
    fireEvent.click(screen.getAllByTitle('Make Spectator')[0]);
    fireEvent.click(screen.getAllByTitle('Make Admin')[0]);
    expect(ws.sent.map((value) => JSON.parse(value).type)).toContain('toggle-spectator');
    expect(ws.sent.map((value) => JSON.parse(value).type)).toContain('toggle-owner');
  });

  it('delays and repeats reminders only for the waiting participant', async () => {
    vi.useFakeTimers();
    renderRoom();
    await act(async () => {});
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({
      participants: [
        { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true },
        { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
        { id: 'third', name: 'Cy', vote: '8', hasVoted: true, isSpectator: false, isOwner: false },
      ],
    })) }));

    act(() => vi.advanceTimersByTime(3999));
    expect(pushSuccess).not.toHaveBeenCalledWith('The room is waiting for your vote.');
    act(() => vi.advanceTimersByTime(1));
    expect(pushSuccess).toHaveBeenCalledTimes(1);
    expect(playNotificationSound).toHaveBeenCalledTimes(1);
    act(() => vi.advanceTimersByTime(20_000));
    expect(pushSuccess).toHaveBeenCalledTimes(3);
    expect(playNotificationSound).toHaveBeenCalledTimes(3);
    act(() => vi.advanceTimersByTime(20_000));
    expect(pushSuccess).toHaveBeenCalledTimes(3);

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({
      participants: [
        { id: 'me', name: 'Ada', vote: '3', hasVoted: true, isSpectator: false, isOwner: true },
        { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
        { id: 'third', name: 'Cy', vote: '8', hasVoted: true, isSpectator: false, isOwner: false },
      ],
    })) }));
    act(() => vi.advanceTimersByTime(20_000));
    expect(pushSuccess).toHaveBeenCalledTimes(3);

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({
      participants: [
        { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true },
        { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
        { id: 'third', name: 'Cy', vote: '8', hasVoted: true, isSpectator: false, isOwner: false },
      ],
    })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(pushSuccess).toHaveBeenCalledTimes(4);
  });

  it('does not remind when two active voters are both waiting', async () => {
    vi.useFakeTimers();
    renderRoom();
    await act(async () => {});
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({
      participants: [
        { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true },
        { id: 'other', name: 'Bob', vote: null, hasVoted: false, isSpectator: false, isOwner: false },
      ],
    })) }));
    act(() => vi.advanceTimersByTime(30_000));
    expect(pushSuccess).not.toHaveBeenCalledWith('The room is waiting for your vote.');
    expect(playNotificationSound).not.toHaveBeenCalled();
  });

  it('honors notification thresholds at the active-voter boundaries', async () => {
    vi.useFakeTimers();
    renderRoom();
    await act(async () => {});
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));

    const meWaiting = { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true };
    const voted = { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false };
    const waiting = { id: 'third', name: 'Cy', vote: null, hasVoted: false, isSpectator: false, isOwner: false };

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [meWaiting, voted] })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledTimes(1);

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [meWaiting, voted, waiting] })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledTimes(2);

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [meWaiting] })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledTimes(2);

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [{ ...meWaiting, isSpectator: true }] })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledTimes(2);

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [{ ...meWaiting, hasVoted: true, vote: '3' }, voted, { ...waiting, hasVoted: true, vote: '8' }] })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledTimes(2);
  });

  it('cancels reminders on disconnect, spectator transition, and reveal', async () => {
    vi.useFakeTimers();
    renderRoom();
    await act(async () => {});
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [
      { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true },
      { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
      { id: 'third', name: 'Cy', vote: '8', hasVoted: true, isSpectator: false, isOwner: false },
    ] })) }));
    act(() => ws.onclose?.({ code: 1006, reason: 'lost' }));
    act(() => vi.advanceTimersByTime(30_000));
    expect(pushSuccess).not.toHaveBeenCalledWith('The room is waiting for your vote.');

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: [
      { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: true, isOwner: true },
      { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
      { id: 'third', name: 'Cy', vote: '8', hasVoted: true, isSpectator: false, isOwner: false },
    ] })) }));
    act(() => vi.advanceTimersByTime(30_000));
    expect(pushSuccess).not.toHaveBeenCalledWith('The room is waiting for your vote.');

    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ reveal: true })) }));
    act(() => vi.advanceTimersByTime(30_000));
    expect(playNotificationSound).not.toHaveBeenCalled();
  });

  it('keeps repeat counts when participants are reordered', async () => {
    vi.useFakeTimers();
    renderRoom();
    await act(async () => {});
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    const participants = [
      { id: 'me', name: 'Ada', vote: null, hasVoted: false, isSpectator: false, isOwner: true },
      { id: 'other', name: 'Bob', vote: '5', hasVoted: true, isSpectator: false, isOwner: false },
      { id: 'third', name: 'Cy', vote: '8', hasVoted: true, isSpectator: false, isOwner: false },
    ];
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledOnce();
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: participants.slice().reverse().map((participant) => participant.id === 'me' ? { ...participant, name: 'Ada Renamed' } : participant) })) }));
    act(() => vi.advanceTimersByTime(10_000));
    expect(playNotificationSound).toHaveBeenCalledTimes(2);
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants })) }));
    act(() => vi.advanceTimersByTime(10_000));
    expect(playNotificationSound).toHaveBeenCalledTimes(3);
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ participants: participants.slice().reverse() })) }));
    act(() => vi.advanceTimersByTime(4000));
    expect(playNotificationSound).toHaveBeenCalledTimes(3);
  });

  it('renders revealed results, stale/kicked/error states, and reconnects', async () => {
    vi.useFakeTimers();
    renderRoom();
    await act(async () => {});
    expect(socketRef.current).not.toBeNull();
    const ws = socketRef.current!;
    act(() => ws.onopen?.());
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ reveal: true, result: 5.5, consensus: 'High', lowestVote: 3, highestVote: 8, voteRange: 5, voteSpread: 2, nonNumericVoteCount: 1, mostAppearingVotes: [5] })) }));
    expect(screen.getByText('Consensus: High')).toBeTruthy();
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'stale-command' }) }));
    act(() => ws.onmessage?.({ data: '{bad json' }));
    expect(pushError).toHaveBeenCalledWith('Room changed; review and try again.');
    expect(pushError).toHaveBeenCalledWith(expect.stringContaining('Error while handling websocket message'));
    act(() => ws.onclose?.({ code: 1006, reason: 'lost' }));
    act(() => vi.advanceTimersByTime(1000));
    expect(socketRef.current?.url).toContain('/planning/123e4567-e89b-12d3-a456-426614174000/ws');
    act(() => socketRef.current?.onmessage?.({ data: JSON.stringify({ type: 'kicked' }) }));
    expect(window.planning_poker?.clientID).toBeUndefined();
    expect(push).toHaveBeenCalledWith('/');
    vi.useRealTimers();
  });

  it('removes socket handlers and closes the socket on unmount', async () => {
    vi.useFakeTimers();
    const view = renderRoom();
    expect(socketRef.current).not.toBeNull();
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    expect(window.planning_poker?.clientID).toBe('me');
    view.unmount();
    expect(window.planning_poker?.clientID).toBeUndefined();
    expect(ws.onopen).toBeNull();
    expect(ws.onmessage).toBeNull();
    expect(ws.onclose).toBeNull();
    expect(ws.onerror).toBeNull();
    expect(ws.readyState).toBe(FakeSocket.CLOSED);
  });

  it('cancels a scheduled reconnect on unmount', async () => {
    vi.useFakeTimers();
    const view = renderRoom();
    expect(socketRef.current).not.toBeNull();
    const ws = socketRef.current!;
    act(() => ws.onclose?.({ code: 1006, reason: 'lost' }));
    view.unmount();
    act(() => vi.runAllTimers());
    expect(socketRef.current).toBeNull();
  });

  it('edits stories and uses backlog actions', async () => {
    renderRoom();
    await waitFor(() => expect(socketRef.current).not.toBeNull());
    const ws = socketRef.current!;
    act(() => ws.onmessage?.({ data: JSON.stringify({ type: 'update-client-id', clientId: 'me' }) }));
    act(() => ws.onmessage?.({ data: JSON.stringify(roomState({ backlogMode: true, stories: [{ id: 's1', name: 'Implement feature', mostAppearingVotes: [], voted: false }, { id: 's2', name: 'Next', mostAppearingVotes: [], voted: false }], currentStoryIndex: 0 })) }));
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: 'Renamed' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(ws.sent.map((value) => JSON.parse(value))).toContainEqual({ type: 'update-story', payload: { story: 'Renamed', expectedRoomVersion: 4 } });
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: 'Changed then cancelled' } });
    fireEvent.keyDown(screen.getByLabelText('Story editor'), { key: 'Escape' });
    expect(screen.getByText('Renamed')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: '' } });
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(false));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(screen.getByLabelText('Story editor')).toBeTruthy();
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(true));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    fireEvent.click(screen.getByRole('button', { name: 'Open backlog' }));
    expect(screen.getByRole('dialog', { name: 'Story Backlog' })).toBeTruthy();
    expect(ws.sent.map((value) => JSON.parse(value).type)).toContain('update-story');
    expect(ws.sent.map((value) => JSON.parse(value))).toContainEqual({ type: 'remove-story', payload: { storyId: 's1', expectedRoomVersion: 4 } });
  });

});
