import type { RoomSnapshot } from '@/hooks/room/roomState';
import { styles } from '../page.styles';

type Props = Readonly<{ snapshot: RoomSnapshot }>;

export default function RoomResultsSummary({ snapshot }: Props) {
  if (!snapshot.reveal) return null;
  const { consensus, result, mostAppearingVotes, lowestVote, highestVote, voteRange, voteSpread, nonNumericVoteCount } = snapshot;
  const showConsensus = consensus !== null && consensus !== 'Unavailable';
  const showAverage = result !== null;
  const showMostCommon = mostAppearingVotes.length > 0;
  const showRange = lowestVote !== null && highestVote !== null;
  const showSpread = voteSpread !== null;
  const showNonNumeric = result !== null && nonNumericVoteCount > 0;
  if (!showConsensus && !showAverage && !showMostCommon && !showRange && !showSpread && !showNonNumeric) return null;

  return <div style={styles.summary}><h4 style={styles.summaryTitle}>Results Summary</h4><div style={styles.summaryContent}>
    {showConsensus && <div>Consensus: {consensus}</div>}
    {showAverage && <div>Average: {result.toFixed(1)}</div>}
    {showMostCommon && <div>Most Common: {mostAppearingVotes.join(', ')}</div>}
    {showRange && <div>Votes range from {lowestVote} to {highestVote} (spread: {voteRange ?? 0})</div>}
    {showSpread && <div>Deck spread: {voteSpread} step{voteSpread === 1 ? '' : 's'}</div>}
    {showNonNumeric && <div>Non-numeric votes: {nonNumericVoteCount}</div>}
  </div></div>;
}
