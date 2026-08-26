import { describe, expect, it } from 'vitest';

import { getExtremeVotes } from './consensus';

describe('getExtremeVotes', () => {
  it('identifies the lowest and highest vote', () => {
    expect(getExtremeVotes('3', 3, 13)).toEqual(['lowest']);
    expect(getExtremeVotes('13', 3, 13)).toEqual(['highest']);
  });

  it('does not identify extremes when all numeric votes are equal', () => {
    expect(getExtremeVotes('5', 5, 5)).toEqual([]);
  });

  it('handles special votes without treating them as numeric extremes', () => {
    expect(getExtremeVotes('?', 3, 13)).toEqual([]);
    expect(getExtremeVotes('☕', 3, 13)).toEqual([]);
  });

  it('handles participants sharing the same extreme value', () => {
    expect(getExtremeVotes('3', 3, 13)).toEqual(['lowest']);
    expect(getExtremeVotes('13', 3, 13)).toEqual(['highest']);
  });
});
