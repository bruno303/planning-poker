const NEUTRAL_LABEL_COLOR = '#64748b';

export function cardColor(card: string | null): string {
  if (card === null) {
    return '#9ca3af';
  }

  if (card === '?') {
    return '#8b5cf6';
  }

  if (card === '☕') {
    return '#f59e0b';
  }

  if (card.trim() === '') {
    return NEUTRAL_LABEL_COLOR;
  }

  const numericCard = Number(card);
  if (!Number.isFinite(numericCard)) {
    return NEUTRAL_LABEL_COLOR;
  }

  if (numericCard <= 2) return '#10b981';
  if (numericCard <= 8) return '#eab308';
  if (numericCard <= 21) return '#f97316';
  return '#ef4444';
}
