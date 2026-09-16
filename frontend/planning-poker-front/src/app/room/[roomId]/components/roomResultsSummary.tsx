import type { RoomSnapshot } from '@/hooks/room/roomState';
import { styles } from '../page.styles';

type Props = Readonly<{ snapshot: RoomSnapshot }>;

export default function RoomResultsSummary({ snapshot }: Props) {
  if (!snapshot.reveal) return null;
  const { consensus, result, mostAppearingVotes, lowestVote, highestVote, voteRange, voteSpread, nonNumericVoteCount } = snapshot;
  return <div style={styles.summary}><h4 style={styles.summaryTitle}>Results Summary</h4><div style={styles.summaryContent}>
    <div>Consensus: {consensus ?? 'Unavailable'}</div><div>Average: {result !== null ? result.toFixed(1) : 'Unavailable'}</div><div>Most Common: {mostAppearingVotes.length ? mostAppearingVotes.join(', ') : 'Unavailable'}</div>
    {lowestVote !== null && highestVote !== null && <div>Votes range from {lowestVote} to {highestVote} (spread: {voteRange ?? 0})</div>}
    {voteSpread !== null && <div>Deck spread: {voteSpread} step{voteSpread === 1 ? '' : 's'}</div>}
    {nonNumericVoteCount > 0 && <div>Non-numeric votes: {nonNumericVoteCount}</div>}
  </div></div>;
}
