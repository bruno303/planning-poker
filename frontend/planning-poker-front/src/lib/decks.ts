export interface DeckPreset {
  id: string;
  name: string;
  cards: string[];
}

export const FALLBACK_DECK: DeckPreset = {
  id: 'fibonacci',
  name: 'Fibonacci',
  cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'],
};

function isDeckPreset(value: unknown): value is DeckPreset {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.name === 'string' &&
    Array.isArray(candidate.cards) &&
    candidate.cards.every((card) => typeof card === 'string')
  );
}

export async function fetchDeckPresets(): Promise<DeckPreset[]> {
  try {
    const res = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/planning/decks`);
    if (!res.ok) {
      return [FALLBACK_DECK];
    }

    const data: unknown = await res.json();
    if (typeof data !== 'object' || data === null || !Array.isArray((data as { decks?: unknown }).decks)) {
      return [FALLBACK_DECK];
    }

    const decks: unknown[] = (data as { decks: unknown[] }).decks;
    if (decks.length === 0 || !decks.every(isDeckPreset)) {
      return [FALLBACK_DECK];
    }

    return decks;
  } catch {
    return [FALLBACK_DECK];
  }
}
