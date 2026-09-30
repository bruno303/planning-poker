import type { RoomSnapshot } from '@/hooks/room/roomState';
import { styles } from '../page.styles';

type Props = Readonly<{ snapshot: RoomSnapshot }>;

export default function RoomResultsSummary({ snapshot }: Props) {
  if (!snapshot.reveal) return null;
  const { consensus, result, mostCommonVotes, lowestVote, highestVote, voteRange, voteSpread, nonNumericVoteCount } = snapshot;
  const isCategorical = snapshot.deck?.kind === 'categorical';
  return <div style={styles.summary}><h4 style={styles.summaryTitle}>Results Summary</h4><div style={styles.summaryContent}>
    <div>Consensus: {consensus ?? 'Unavailable'}</div>
    {!isCategorical && <div>Average: {result !== null ? result.toFixed(1) : 'Unavailable'}</div>}
    <div>Most Common: {mostCommonVotes.length ? mostCommonVotes.join(', ') : 'Unavailable'}</div>
    {!isCategorical && lowestVote !== null && highestVote !== null && <div>Votes range from {lowestVote} to {highestVote} (spread: {voteRange ?? 0})</div>}
    {!isCategorical && voteSpread !== null && <div>Deck spread: {voteSpread} step{voteSpread === 1 ? '' : 's'}</div>}
    {!isCategorical && nonNumericVoteCount > 0 && <div>Non-numeric votes: {nonNumericVoteCount}</div>}
  </div></div>;
}
