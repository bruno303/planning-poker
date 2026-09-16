import Avatar from '@/components/avatar/avatar';
import ParticipantIdBadge from '@/components/participantIdBadge/participantIdBadge';
import { getExtremeVotes } from '@/components/consensus/consensus';
import { formatVotedAt } from '@/components/roomClock/roomClock';
import type { Participant, RoomSnapshot } from '@/hooks/room/roomState';
import { Eye, Shield, Users } from 'lucide-react';
import { styles } from '../page.styles';

type Props = { snapshot: RoomSnapshot; amIAdmin: boolean; onToggleSpectator: (id: string) => void; onToggleAdmin: (id: string) => void; onCopied: () => void };
const voteColor = (vote: string | null) => { if (vote === '?') return '#8b5cf6'; if (vote === '☕') return '#f59e0b'; const n = Number(vote); return n <= 2 ? '#10b981' : n <= 8 ? '#eab308' : n <= 21 ? '#f97316' : '#ef4444'; };
const status = (p: Participant) => p.isSpectator ? 'Spectator' : p.hasVoted ? 'Voted' : 'Waiting...';

export default function RoomParticipantsPanel({ snapshot, amIAdmin, onToggleSpectator, onToggleAdmin, onCopied }: Props) {
  const styleFor = (p: Participant, extremes: string[]) => ({ ...styles.participant, ...(p.isSpectator ? styles.participantSpectator : p.hasVoted ? styles.participantVoted : styles.participantWaiting), ...(extremes.includes('lowest') ? styles.participantLowest : extremes.includes('highest') ? styles.participantHighest : {}) });
  return <><div style={styles.participantsHeader}><Users color="#3b82f6" size={24} /><h3 style={styles.sectionTitle}>Participants</h3></div><div style={styles.participantsList}>{snapshot.participants.map((participant) => {
    const extremes = snapshot.reveal && !participant.isSpectator ? getExtremeVotes(participant.vote, snapshot.lowestVote, snapshot.highestVote) : [];
    const votedAt = participant.isSpectator || !participant.hasVoted || participant.vote === null ? null : formatVotedAt(participant.votedAt ?? undefined, snapshot.startedAt);
    return <div key={participant.id} style={styleFor(participant, extremes)}><div style={styles.participantContent}><div><div style={styles.participantName}><Avatar participant={participant} />{participant.name}{amIAdmin && <ParticipantIdBadge participantId={participant.id} onCopied={onCopied} />}{extremes.map((extreme) => <span key={extreme} style={extreme === 'lowest' ? styles.lowestVoteBadge : styles.highestVoteBadge}>{extreme === 'lowest' ? 'Lowest' : 'Highest'}</span>)}</div><div style={styles.participantStatus}>{status(participant)}</div>{votedAt && <div style={styles.participantVotedAt}>Voted at {votedAt}</div>}</div><div style={styles.participantRight}>{!participant.isSpectator && participant.hasVoted && <div style={{ ...styles.voteCard, backgroundColor: snapshot.reveal ? voteColor(participant.vote) : '#9ca3af' }}>{snapshot.reveal ? participant.vote : '?'}</div>}{amIAdmin && <div style={styles.adminControls}><button onClick={() => onToggleSpectator(participant.id)} style={{ ...styles.roleButton, ...(participant.isSpectator ? styles.activeSpectatorButton : styles.inactiveButton) }} title={participant.isSpectator ? 'Make Voter' : 'Make Spectator'}><Eye size={12} /></button><button onClick={() => onToggleAdmin(participant.id)} style={{ ...styles.roleButton, ...(participant.isOwner ? styles.activeAdminButton : styles.inactiveButton) }} title={participant.isOwner ? 'Remove Admin' : 'Make Admin'}><Shield size={12} /></button></div>}<div style={{ ...styles.statusDot, ...(participant.isSpectator ? styles.statusSpectator : participant.hasVoted ? styles.statusVoted : styles.statusWaiting) }} /></div></div></div>;
  })}</div></>;
}
