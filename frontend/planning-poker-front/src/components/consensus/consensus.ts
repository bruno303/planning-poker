export type ExtremeVote = 'lowest' | 'highest';

export function getExtremeVotes(
  vote: string | null,
  lowestVote: number | null,
  highestVote: number | null,
): ExtremeVote[] {
  if (vote === null || lowestVote === null || highestVote === null || lowestVote === highestVote) {
    return [];
  }

  const trimmedVote = vote.trim();
  if (!/^[+-]?\d+$/.test(trimmedVote)) {
    return [];
  }

  const numericVote = Number(trimmedVote);
  const extremes: ExtremeVote[] = [];
  if (numericVote === lowestVote) {
    extremes.push('lowest');
  }
  if (numericVote === highestVote) {
    extremes.push('highest');
  }

  return extremes;
}
