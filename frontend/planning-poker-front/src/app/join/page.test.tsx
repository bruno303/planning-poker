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

const deckPresets = [
  { id: 'fibonacci', name: 'Fibonacci', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] },
  { id: 'tshirt', name: 'T-shirt sizes', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] },
];

const decksResult = { ok: true, json: async () => ({ decks: deckPresets }) };
const createResult = { ok: true, json: async () => ({ roomId: 'new room' }) };

const stubFetch = (overrides: { decks?: () => Promise<unknown>; post?: () => Promise<unknown> } = {}) => {
  const fetchMock = vi.fn((_: string, init?: RequestInit) => {
    if (init?.method === 'POST') {
      return overrides.post ? overrides.post() : Promise.resolve(createResult);
    }
    return overrides.decks ? overrides.decks() : Promise.resolve(decksResult);
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
};

const postCall = (fetchMock: ReturnType<typeof stubFetch>) => {
  const call = fetchMock.mock.calls.find(([, init]) => (init as RequestInit | undefined)?.method === 'POST');
  return call?.[1] as RequestInit | undefined;
};

describe('join page', () => {
  afterEach(() => cleanup());
  beforeEach(() => {
    params = {};
    push.mockReset();
    pushError.mockReset();
    logger.warn.mockReset();
    vi.restoreAllMocks();
    sessionStorage.clear();
  });

  it('validates name before creating a room', async () => {
    stubFetch();
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.keyDown(screen.getByLabelText('Your Name'), { key: 'Enter' });
    expect(pushError).toHaveBeenCalledWith('Name not informed');
  });

  it('creates a room, stores the trimmed name, and navigates', async () => {
    stubFetch();
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: ' Ada ' } });
    fireEvent.click(screen.getAllByRole('button', { name: /create room/i })[0]);
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/new%20room'));
    expect(sessionStorage.getItem('userName')).toBe('Ada');
  });

  it('loads the deck presets and defaults to the first one', async () => {
    stubFetch();
    render(<Home />);

    await screen.findByRole('option', { name: 'T-shirt sizes' });
    expect(screen.getByRole('option', { name: 'Fibonacci' })).toBeTruthy();
    expect((screen.getByLabelText('Voting deck') as HTMLSelectElement).value).toBe('fibonacci');
  });

  it('creates a room with the selected deck preset', async () => {
    const fetchMock = stubFetch();
    render(<Home />);
    await screen.findByRole('option', { name: 'T-shirt sizes' });
    fireEvent.change(screen.getByLabelText('Voting deck'), { target: { value: 'tshirt' } });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getAllByRole('button', { name: /create room/i })[0]);

    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/new%20room'));
    const request = postCall(fetchMock);
    expect(request?.headers).toEqual({ 'Content-Type': 'application/json' });
    expect(JSON.parse(request?.body as string)).toEqual({ deckPreset: 'tshirt' });
  });

  it('falls back to the Fibonacci deck when the presets request fails', async () => {
    const fetchMock = stubFetch({ decks: () => Promise.reject(new Error('server down')) });
    render(<Home />);

    await screen.findByRole('option', { name: 'Fibonacci' });
    expect((screen.getByLabelText('Voting deck') as HTMLSelectElement).value).toBe('fibonacci');
    expect(logger.warn).toHaveBeenCalledWith(expect.stringContaining('fallback'));

    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getAllByRole('button', { name: /create room/i })[0]);
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/new%20room'));
    expect(JSON.parse(postCall(fetchMock)?.body as string)).toEqual({ deckPreset: 'fibonacci' });
  });

  it('does not render the deck picker when joining an existing room', () => {
    params = { roomId: 'room-1' };
    stubFetch();
    render(<Home />);

    expect(screen.queryByLabelText('Voting deck')).toBeNull();
  });

  it('shows the create loading state and disables the button while pending', async () => {
    let resolveCreate: (value: typeof createResult) => void = () => {};
    stubFetch({ post: () => new Promise((res) => { resolveCreate = res; }) });
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    const button = screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement;
    fireEvent.click(button);
    expect(screen.getByText('Creating Room...')).toBeTruthy();
    expect(button.disabled).toBe(true);
    resolveCreate(createResult);
    await waitFor(() => expect(push).toHaveBeenCalled());
  });

  it('reports a non-OK create response', async () => {
    stubFetch({ post: () => Promise.resolve({ ok: false }) });
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('Failed to create room on server'));
  });

  it('reports create failures and returns to an enabled button', async () => {
    stubFetch({ post: () => Promise.reject(new Error('server down')) });
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('server down'));
    expect((screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement).disabled).toBe(false);
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

  it('shows the join loading state and disables the button while navigation is pending', () => {
    params = { roomId: 'room-1' };
    push.mockImplementation(() => new Promise<void>(() => {}));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Bob' } });
    const button = screen.getByRole('button', { name: /join room/i }) as HTMLButtonElement;
    fireEvent.click(button);
    expect(screen.getByText('Joining Room...')).toBeTruthy();
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

  it('keeps create disabled until a name is supplied', async () => {
    params = {};
    stubFetch();
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    expect((screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement).disabled).toBe(true);
  });
});
