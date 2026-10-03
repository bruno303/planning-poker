import type { RoomSnapshot } from '@/hooks/room/roomState';
import { styles } from '../page.styles';

type Props = Readonly<{ snapshot: RoomSnapshot }>;

export default function RoomResultsSummary({ snapshot }: Props) {
  if (!snapshot.reveal) return null;
  const { consensus, deck, result, mostAppearingVotes, lowestVote, highestVote, voteRange, voteSpread, specialVoteCount } = snapshot;
  const showAverage = deck.id === 'fibonacci' || result !== null;
  return <div style={styles.summary}><h4 style={styles.summaryTitle}>Results Summary</h4><div style={styles.summaryContent}>
    <div>Consensus: {consensus ?? 'Unavailable'}</div>
    {showAverage && <div>Average: {result !== null ? result.toFixed(1) : 'Unavailable'}</div>}
    <div>Most Common: {mostAppearingVotes.length ? mostAppearingVotes.join(', ') : 'Unavailable'}</div>
    {lowestVote !== null && highestVote !== null && <div>Votes range from {lowestVote} to {highestVote}{voteRange !== null ? ` (spread: ${voteRange})` : ''}</div>}
    {voteSpread !== null && <div>Deck spread: {voteSpread} step{voteSpread === 1 ? '' : 's'}</div>}
    {specialVoteCount > 0 && <div>Special votes: {specialVoteCount}</div>}
  </div></div>;
}
