import { afterEach, describe, expect, it, vi } from 'vitest';

import { FALLBACK_DECK, fetchDeckPresets, type DeckPreset } from './decks';

const decks: DeckPreset[] = [
  { id: 'fibonacci', name: 'Fibonacci', cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'] },
  { id: 'tshirt', name: 'T-shirt sizes', cards: ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'] },
];

describe('fetchDeckPresets', () => {
  afterEach(() => vi.restoreAllMocks());

  it('returns the deck presets from the backend', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ decks }) });
    vi.stubGlobal('fetch', fetchMock);

    await expect(fetchDeckPresets()).resolves.toEqual(decks);
    expect(fetchMock).toHaveBeenCalledWith(`${process.env.NEXT_PUBLIC_BACKEND_URL}/planning/decks`);
  });

  it('falls back to the Fibonacci deck when the request fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network down')));

    await expect(fetchDeckPresets()).resolves.toEqual([FALLBACK_DECK]);
  });

  it('falls back to the Fibonacci deck on a non-OK response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }));

    await expect(fetchDeckPresets()).resolves.toEqual([FALLBACK_DECK]);
  });

  it.each([
    ['missing decks key', {}],
    ['decks not an array', { decks: 'fibonacci' }],
    ['empty decks', { decks: [] }],
    ['malformed deck', { decks: [{ id: 'fibonacci', name: 'Fibonacci' }] }],
    ['non-string cards', { decks: [{ id: 'fibonacci', name: 'Fibonacci', cards: ['0', 1] }] }],
    ['null payload', null],
  ])('falls back to the Fibonacci deck for an invalid payload (%s)', async (_, payload) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => payload }));

    await expect(fetchDeckPresets()).resolves.toEqual([FALLBACK_DECK]);
  });

  it('falls back to the Fibonacci deck when the payload cannot be parsed', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => { throw new Error('invalid json'); } }));

    await expect(fetchDeckPresets()).resolves.toEqual([FALLBACK_DECK]);
  });
});
