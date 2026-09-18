import Avatar from '@/components/avatar/avatar';
import ParticipantIdBadge from '@/components/participantIdBadge/participantIdBadge';
import { getExtremeVotes } from '@/components/consensus/consensus';
import { formatVotedAt } from '@/components/roomClock/roomClock';
import type { Participant, RoomSnapshot } from '@/hooks/room/roomState';
import { Eye, Shield, Users } from 'lucide-react';
import { styles } from '../page.styles';

type Props = Readonly<{ snapshot: RoomSnapshot; amIAdmin: boolean; onToggleSpectator: (id: string) => void; onToggleAdmin: (id: string) => void; onCopied: () => void }>;

const voteColor = (vote: string | null) => {
  if (vote === '?') return '#8b5cf6';
  if (vote === '☕') return '#f59e0b';
  const numericVote = Number(vote);
  if (numericVote <= 2) return '#10b981';
  if (numericVote <= 8) return '#eab308';
  if (numericVote <= 21) return '#f97316';
  return '#ef4444';
};

const status = (participant: Participant) => {
  if (participant.isSpectator) return 'Spectator';
  if (participant.hasVoted) return 'Voted';
  return 'Waiting...';
};

const extremeStyle = (extreme: string) => {
  if (extreme === 'lowest') return styles.lowestVoteBadge;
  return styles.highestVoteBadge;
};

const extremeLabel = (extreme: string) => {
  if (extreme === 'lowest') return 'Lowest';
  return 'Highest';
};

export default function RoomParticipantsPanel({ snapshot, amIAdmin, onToggleSpectator, onToggleAdmin, onCopied }: Props) {
  const styleFor = (participant: Participant, extremes: string[]) => {
    let participantState = styles.participantWaiting;
    if (participant.isSpectator) participantState = styles.participantSpectator;
    else if (participant.hasVoted) participantState = styles.participantVoted;

    let extremeState = {};
    if (extremes.includes('lowest')) extremeState = styles.participantLowest;
    else if (extremes.includes('highest')) extremeState = styles.participantHighest;
    return { ...styles.participant, ...participantState, ...extremeState };
  };

  return <><div style={styles.participantsHeader}><Users color="#3b82f6" size={24} /><h3 style={styles.sectionTitle}>Participants</h3></div><div style={styles.participantsList}>{snapshot.participants.map((participant) => {
    const extremes = snapshot.reveal && !participant.isSpectator ? getExtremeVotes(participant.vote, snapshot.lowestVote, snapshot.highestVote) : [];
    const votedAt = participant.isSpectator || !participant.hasVoted || participant.vote === null ? null : formatVotedAt(participant.votedAt ?? undefined, snapshot.startedAt);
    let participantStatus = styles.statusWaiting;
    if (participant.isSpectator) participantStatus = styles.statusSpectator;
    else if (participant.hasVoted) participantStatus = styles.statusVoted;
    return <div key={participant.id} style={styleFor(participant, extremes)}><div style={styles.participantContent}><div><div style={styles.participantName}><Avatar participant={participant} />{participant.name}{amIAdmin && <ParticipantIdBadge participantId={participant.id} onCopied={onCopied} />}{extremes.map((extreme) => <span key={extreme} style={extremeStyle(extreme)}>{extremeLabel(extreme)}</span>)}</div><div style={styles.participantStatus}>{status(participant)}</div>{votedAt && <div style={styles.participantVotedAt}>Voted at {votedAt}</div>}</div><div style={styles.participantRight}>{!participant.isSpectator && participant.hasVoted && <div style={{ ...styles.voteCard, backgroundColor: snapshot.reveal ? voteColor(participant.vote) : '#9ca3af' }}>{snapshot.reveal ? participant.vote : '?'}</div>}{amIAdmin && <div style={styles.adminControls}><button onClick={() => onToggleSpectator(participant.id)} style={{ ...styles.roleButton, ...(participant.isSpectator ? styles.activeSpectatorButton : styles.inactiveButton) }} title={participant.isSpectator ? 'Make Voter' : 'Make Spectator'}><Eye size={12} /></button><button onClick={() => onToggleAdmin(participant.id)} style={{ ...styles.roleButton, ...(participant.isOwner ? styles.activeAdminButton : styles.inactiveButton) }} title={participant.isOwner ? 'Remove Admin' : 'Make Admin'}><Shield size={12} /></button></div>}<div style={{ ...styles.statusDot, ...participantStatus }} /></div></div></div>;
  })}</div></>;
}
