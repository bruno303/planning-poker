import { describe, expect, it } from 'vitest';

import { getCardColor, getExtremeVotes } from './consensus';

const fibonacciCards = ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'];
const tShirtCards = ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'];

describe('getExtremeVotes', () => {
  it('identifies the lowest and highest label', () => {
    expect(getExtremeVotes('3', '3', '13')).toEqual(['lowest']);
    expect(getExtremeVotes('13', '3', '13')).toEqual(['highest']);
  });

  it('supports ordered T-shirt labels', () => {
    expect(getExtremeVotes('XS', 'XS', 'XL')).toEqual(['lowest']);
    expect(getExtremeVotes('XL', 'XS', 'XL')).toEqual(['highest']);
    expect(getExtremeVotes('M', 'XS', 'XL')).toEqual([]);
  });

  it('does not identify extremes when all votes share the same label', () => {
    expect(getExtremeVotes('5', '5', '5')).toEqual([]);
  });

  it('handles special votes without treating them as extremes', () => {
    expect(getExtremeVotes('?', '3', '13')).toEqual([]);
    expect(getExtremeVotes('☕', '3', '13')).toEqual([]);
  });

  it('handles missing votes and labels', () => {
    expect(getExtremeVotes(null, '3', '13')).toEqual([]);
    expect(getExtremeVotes('3', null, '13')).toEqual([]);
    expect(getExtremeVotes('3', '3', null)).toEqual([]);
  });

  it('handles participants sharing the same extreme value', () => {
    expect(getExtremeVotes('3', '3', '13')).toEqual(['lowest']);
    expect(getExtremeVotes('13', '3', '13')).toEqual(['highest']);
  });
});

describe('getCardColor', () => {
  it('derives Fibonacci colors from ordered positions', () => {
    expect(getCardColor('0', fibonacciCards)).toBe('#10b981');
    expect(getCardColor('3', fibonacciCards)).toBe('#eab308');
    expect(getCardColor('8', fibonacciCards)).toBe('#eab308');
    expect(getCardColor('13', fibonacciCards)).toBe('#f97316');
    expect(getCardColor('21', fibonacciCards)).toBe('#f97316');
    expect(getCardColor('34', fibonacciCards)).toBe('#ef4444');
    expect(getCardColor('89', fibonacciCards)).toBe('#ef4444');
  });

  it('derives T-shirt colors from ordered positions', () => {
    expect(getCardColor('XS', tShirtCards)).toBe('#10b981');
    expect(getCardColor('S', tShirtCards)).toBe('#10b981');
    expect(getCardColor('M', tShirtCards)).toBe('#eab308');
    expect(getCardColor('L', tShirtCards)).toBe('#f97316');
    expect(getCardColor('XL', tShirtCards)).toBe('#ef4444');
    expect(getCardColor('XXL', tShirtCards)).toBe('#ef4444');
  });

  it('keeps explicit special-card colors and a safe fallback', () => {
    expect(getCardColor('?', tShirtCards)).toBe('#8b5cf6');
    expect(getCardColor('☕', fibonacciCards)).toBe('#f59e0b');
    expect(getCardColor(null, fibonacciCards)).toBe('#6b7280');
    expect(getCardColor('99', fibonacciCards)).toBe('#6b7280');
  });
});
