import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Home from './page';

const push = vi.fn();
const pushError = vi.fn();
const logger = { warn: vi.fn(), info: vi.fn(), error: vi.fn(), setContext: vi.fn() };
let params: { roomId?: string } = {};

const deckCatalogue = {
  decks: [
    { id: 'fibonacci', name: 'Fibonacci', kind: 'numeric', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] },
    { id: 't-shirt', name: 'T-shirt', kind: 'categorical', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] },
  ],
};

type StubbedResponse = { ok: boolean; json: () => Promise<unknown> };

const stubFetch = (createResponse?: (init?: RequestInit) => StubbedResponse | Promise<StubbedResponse>) => {
  const mock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    if (String(input).endsWith('/planning/decks')) {
      return Promise.resolve({ ok: true, json: async () => deckCatalogue });
    }
    if (createResponse) {
      return Promise.resolve(createResponse(init));
    }
    return Promise.resolve({ ok: true, json: async () => ({ roomId: 'room' }) });
  });
  vi.stubGlobal('fetch', mock);
  return mock;
};

const createRoomRequest = (mock: ReturnType<typeof stubFetch>) =>
  mock.mock.calls.find(([input, init]) => String(input).endsWith('/planning/rooms') && (init as RequestInit | undefined)?.method === 'POST');

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

  it('validates name before creating a room', async () => {
    stubFetch();
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.keyDown(screen.getByLabelText('Your Name'), { key: 'Enter' });
    expect(pushError).toHaveBeenCalledWith('Name not informed');
  });

  it('loads the deck catalogue and creates a room with the default deck', async () => {
    const fetchMock = stubFetch(() => ({ ok: true, json: async () => ({ roomId: 'new room' }) }));
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: ' Ada ' } });
    fireEvent.click(screen.getAllByRole('button', { name: /create room/i })[0]);
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/new%20room'));
    expect(sessionStorage.getItem('userName')).toBe('Ada');

    const request = createRoomRequest(fetchMock);
    expect(request).toBeDefined();
    expect(JSON.parse((request?.[1] as RequestInit).body as string)).toEqual({ deckType: 'fibonacci' });
    expect((request?.[1] as RequestInit).headers).toEqual({ 'Content-Type': 'application/json' });
  });

  it('submits the selected T-shirt deck type', async () => {
    const fetchMock = stubFetch(() => ({ ok: true, json: async () => ({ roomId: 'tshirt-room' }) }));
    render(<Home />);
    await screen.findByRole('option', { name: 'T-shirt' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.change(screen.getByLabelText('Deck'), { target: { value: 't-shirt' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/tshirt-room'));

    const request = createRoomRequest(fetchMock);
    expect(JSON.parse((request?.[1] as RequestInit).body as string)).toEqual({ deckType: 't-shirt' });
  });

  it('shows the create loading state and disables the button while pending', async () => {
    let resolve: (value: StubbedResponse) => void = () => {};
    stubFetch(() => new Promise((res) => { resolve = res; }));
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    const button = screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement;
    fireEvent.click(button);
    expect(screen.getByText('Creating Room...')).toBeTruthy();
    expect(button.disabled).toBe(true);
    resolve({ ok: true, json: async () => ({ roomId: 'room' }) });
    await waitFor(() => expect(push).toHaveBeenCalled());
  });

  it('reports a non-OK create response', async () => {
    stubFetch(() => ({ ok: false, json: async () => ({}) }));
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('Failed to create room on server'));
  });

  it('reports create failures and returns to an enabled button', async () => {
    stubFetch(() => Promise.reject(new Error('server down')));
    render(<Home />);
    await screen.findByRole('option', { name: 'Fibonacci' });
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('server down'));
    expect((screen.getAllByRole('button', { name: /create room/i })[0] as HTMLButtonElement).disabled).toBe(false);
  });

  it('reports catalogue failures and keeps create disabled', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }));
    render(<Home />);
    await waitFor(() => expect(pushError).toHaveBeenCalledWith('Failed to load deck catalogue'));
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Ada' } });
    expect((screen.getByRole('button', { name: /create room/i }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('rejects deck catalogue entries with an unknown kind', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ decks: [{ ...deckCatalogue.decks[0], kind: 'ordinal' }] }),
    }));
    render(<Home />);

    await waitFor(() => expect(pushError).toHaveBeenCalledWith('Invalid deck catalogue'));
  });

  it('joins a room from the route and submits on Enter', async () => {
    params = { roomId: 'room 1' };
    const fetchMock = stubFetch();
    render(<Home />);
    const input = screen.getByLabelText('Your Name');
    fireEvent.change(input, { target: { value: ' Bob ' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    await waitFor(() => expect(push).toHaveBeenCalledWith('/room/room%201'));
    expect(sessionStorage.getItem('userName')).toBe('Bob');
    expect(screen.queryByLabelText('Deck')).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('shows the join loading state and disables the button while navigation is pending', async () => {
    params = { roomId: 'room-1' };
    stubFetch();
    let resolve: () => void = () => {};
    push.mockImplementation(() => new Promise<void>((res) => { resolve = res; }));
    render(<Home />);
    fireEvent.change(screen.getByLabelText('Your Name'), { target: { value: 'Bob' } });
    const button = screen.getByRole('button', { name: /join room/i }) as HTMLButtonElement;
    fireEvent.click(button);
    expect(screen.getByText('Joining Room...')).toBeTruthy();
    expect(button.disabled).toBe(true);
    resolve();
  });

  it('reports join failures', async () => {
    params = { roomId: 'room-1' };
    stubFetch();
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
