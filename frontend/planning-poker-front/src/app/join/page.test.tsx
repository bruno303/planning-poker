import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Home from './page';

const push = vi.fn();
const pushError = vi.fn();
const logger = { warn: vi.fn(), info: vi.fn(), error: vi.fn(), setContext: vi.fn() };
let params: { roomId?: string } = {};

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
  useParams: () => params,
}));
vi.mock('@/context/logger/loggerContext', () => ({ useLogger: () => logger }));
vi.mock('@/context/toast/toastContext', () => ({ useToast: () => ({ pushError }) }));

describe('join page', () => {
  afterEach(() => cleanup());
  beforeEach(() => {
    params = {};
    push.mockReset();
    pushError.mockReset();
    vi.restoreAllMocks();
    sessionStorage.clear();
  });

  it('validates name before creating a room', () => {
    render(<Home />);
    fireEvent.keyDown(screen.getByLabelText('Your Name'), { key: 'Enter' });
    expect(pushError).toHaveBeenCalledWith('Name not informed');
  });

  it('creates a room, stores the trimmed name, and navigates', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ roomId: 'new room' }) }));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: ' Ada ' } });
    fireEvent.click(screen.getAllByRole('button', { name: /create room/i })[0]);
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/new%20room'));
    expect(sessionStorage.getItem('userName')).toBe('Ada');
  });

  it('shows the create loading state and disables the button while pending', async () => {
    let resolve: (value: { ok: boolean; json: () => Promise<{ roomId: string }> }) => void = () => {};
    vi.stubGlobal('fetch', vi.fn(() => new Promise((res) => { resolve = res; })));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    const button = screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement;
    fireEvent.click(button);
    expect(screen.getByText('Creating Room...')).toBeTruthy();
    expect(button.disabled).toBe(true);
    resolve({ ok: true, json: async () => ({ roomId: 'room' }) });
    await waitFor(() => expect(push).toHaveBeenCalled());
  });

  it('reports a non-OK create response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('Failed to create room on server'));
  });

  it('reports create failures and returns to an enabled button', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('server down')));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('server down'));
    expect((screen.getAllByRole('button', { name: /create room/i })[0] as HTMLButtonElement).disabled).toBe(false);
  });

  it('joins a room from the route and submits on Enter', async () => {
    params = { roomId: 'room 1' };
    render(<Home />);
    const input = screen.getByLabelText('Your Name');
    fireEvent.change(input, { target: { value: ' Bob ' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/room%201'));
    expect(sessionStorage.getItem('userName')).toBe('Bob');
  });

  it('shows the join loading state and disables the button while navigation is pending', async () => {
    params = { roomId: 'room-1' };
    let resolve: () => void = () => {};
    push.mockImplementation(() => new Promise<void>((res) => { resolve = res; }));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Bob' } });
    const button = screen.getByRole('button', { name: /join room/i }) as HTMLButtonElement;
    fireEvent.click(button);
    expect(screen.getByText('Joining Room...')).toBeTruthy();
    expect(button.disabled).toBe(true);
    expect(button.disabled).toBe(true);
  });

  it('reports join failures', async () => {
    params = { roomId: 'room-1' };
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('storage unavailable'); });
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Bob' } });
    fireEvent.click(screen.getByRole('button', { name: /join room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('storage unavailable'));
  });

  it('keeps create disabled until a name is supplied', () => {
    params = {};
    render(<Home />);
    expect((screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement).disabled).toBe(true);
  });
});
