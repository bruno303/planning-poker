import { describe, expect, it } from 'vitest';

import { cardColor } from './cardColor';

describe('cardColor', () => {
  it('uses the numeric bands for numeric cards', () => {
    expect(cardColor('0')).toBe('#10b981');
    expect(cardColor('2')).toBe('#10b981');
    expect(cardColor('3')).toBe('#eab308');
    expect(cardColor('8')).toBe('#eab308');
    expect(cardColor('13')).toBe('#f97316');
    expect(cardColor('21')).toBe('#f97316');
    expect(cardColor('34')).toBe('#ef4444');
  });

  it('uses dedicated colors for the special cards', () => {
    expect(cardColor('?')).toBe('#8b5cf6');
    expect(cardColor('☕')).toBe('#f59e0b');
  });

  it('uses the neutral label color for non-numeric labels', () => {
    expect(cardColor('XS')).toBe('#64748b');
    expect(cardColor('')).toBe('#64748b');
    expect(cardColor('   ')).toBe('#64748b');
  });

  it('uses the neutral color when no vote is selected', () => {
    expect(cardColor(null)).toBe('#9ca3af');
  });
});
