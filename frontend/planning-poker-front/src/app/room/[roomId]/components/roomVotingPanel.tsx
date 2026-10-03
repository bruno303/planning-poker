import Avatar from '@/components/avatar/avatar';
import { getCardColor } from '@/components/consensus/consensus';
import { type Card, type RoomSnapshot } from '@/hooks/room/roomState';
import { Eye, EyeOff, List, Repeat, X } from 'lucide-react';
import { styles } from '../page.styles';

type Props = Readonly<{ snapshot: RoomSnapshot; clientId: string; userName: string; amIAdmin: boolean; onSelectCard: (card: Card) => void; onReveal: () => void; onVoteAgain: () => void; onToggleBacklog: () => void }>;

export default function RoomVotingPanel({ snapshot, clientId, userName, amIAdmin, onSelectCard, onReveal, onVoteAgain, onToggleBacklog }: Props) {
  const selected = snapshot.participants.find((participant) => participant.id === clientId)?.vote ?? null;
  const voters = snapshot.participants.filter((participant) => !participant.isSpectator);
  const cards = snapshot.deck.cards;
  const color = (card: Card) => getCardColor(card, cards);
  return <div style={styles.card}><div style={styles.userInfo}><div style={styles.inputGroup}><div style={styles.userNameRow}><Avatar participant={{ id: clientId }} /><label style={{ ...styles.label, marginBottom: 0 }}>{userName}</label></div></div><div style={styles.voteStats}><div style={styles.voteStatsLabel}>Votes Cast</div><div style={styles.voteStatsNumber}>{voters.filter((p) => p.hasVoted).length}/{voters.length}</div></div></div>
    <div style={styles.selectedCard}><div style={styles.selectedCardLabel}>{selected ? 'Your Vote' : 'No Vote Yet'}</div><div style={{ ...styles.selectedCardDisplay, backgroundColor: selected ? color(selected) : '#9ca3af' }}>{selected ?? <X size={32} strokeWidth={3} />}</div></div>
    <div style={{ marginTop: '2rem' }}><h3 style={styles.sectionTitle}>Select Your Card</h3><div style={styles.cardsGrid}>{cards.map((card) => <button key={card} onClick={() => !snapshot.reveal && onSelectCard(card)} aria-disabled={snapshot.reveal} aria-pressed={selected === card} style={{ ...styles.pokerCard, ...(snapshot.reveal ? styles.pokerCardDisabled : {}), backgroundColor: snapshot.reveal ? '#9ca3af' : color(card), ...(selected === card ? styles.pokerCardSelected : {}) }} onMouseEnter={(event) => { if (!snapshot.reveal && selected !== card) (event.target as HTMLButtonElement).style.transform = 'scale(1.05)'; }} onMouseLeave={(event) => { if (!snapshot.reveal && selected !== card) (event.target as HTMLButtonElement).style.transform = 'scale(1)'; }}>{card}</button>)}</div></div>
    {amIAdmin && <div style={styles.buttonsContainer}>{!snapshot.backlogMode && <button onClick={onToggleBacklog} style={{ ...styles.button, ...styles.primaryButton }} onMouseEnter={(event) => (event.target as HTMLButtonElement).style.backgroundColor = '#2563eb'} onMouseLeave={(event) => (event.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}><List size={20} />Enable Backlog</button>}<button onClick={onReveal} style={{ ...styles.button, ...styles.primaryButton }} onMouseEnter={(event) => (event.target as HTMLButtonElement).style.backgroundColor = '#2563eb'} onMouseLeave={(event) => (event.target as HTMLButtonElement).style.backgroundColor = '#3b82f6'}>{snapshot.reveal ? <EyeOff size={20} /> : <Eye size={20} />}{snapshot.reveal ? 'Hide Votes' : 'Reveal Votes'}</button><button onClick={onVoteAgain} style={{ ...styles.button, ...styles.warningButton }} onMouseEnter={(event) => (event.target as HTMLButtonElement).style.backgroundColor = '#f97316'} onMouseLeave={(event) => (event.target as HTMLButtonElement).style.backgroundColor = '#eab308'}><Repeat size={20} />Vote Again</button></div>}
  </div>;
}
