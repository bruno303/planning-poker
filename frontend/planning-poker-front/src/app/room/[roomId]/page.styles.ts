export const styles = {
  container: {
    minHeight: '100vh',
    background: 'linear-gradient(135deg, #dbeafe 0%, #e0e7ff 100%)',
    padding: '1rem',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
  },
  maxWidth: {
    maxWidth: '1280px',
    margin: '0 auto'
  },
  header: {
    textAlign: 'center' as const,
    marginBottom: '2rem'
  },
  title: {
    fontSize: '2.25rem',
    fontWeight: 'bold',
    color: '#1f2937',
    marginBottom: '1rem'
  },
  storyCard: {
    backgroundColor: 'white',
    borderRadius: '0.5rem',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
    padding: '1rem',
    marginBottom: '1rem'
  },
  storyTitle: {
    fontSize: '1.25rem',
    fontWeight: '600',
    color: '#374151',
    margin: 0
  },
  storyHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '0.75rem',
    position: 'relative' as const,
    marginBottom: '0.5rem'
  },
  storyEditButton: {
    padding: '0.5rem 1rem',
    fontSize: '0.875rem',
    position: 'absolute' as const,
    right: 0
  },
  storyText: {
    color: '#6b7280',
    fontStyle: 'italic'
  },
  storyControls: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: '0.75rem',
    flexWrap: 'wrap' as const,
    gridColumn: 3,
    justifySelf: 'end'
  },
  storyLine: {
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1fr) minmax(0, auto) minmax(0, 1fr)',
    alignItems: 'center',
    gap: '0.75rem',
    minWidth: 0
  },
  storyContent: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '0.5rem',
    gridColumn: 2,
    width: '100%',
    minWidth: 0
  },
  storyNavigation: {
    display: 'flex',
    gap: '0.375rem'
  },
  storyControlButton: {
    width: '2.25rem',
    height: '2.25rem',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 0,
    borderRadius: '0.375rem',
    backgroundColor: '#3b82f6',
    color: 'white',
    border: 'none',
    cursor: 'pointer',
    transition: 'background-color 0.2s'
  },
  storyBacklogButton: {
    padding: '0.5rem 0.875rem',
    fontSize: '0.875rem'
  },
  card: {
    backgroundColor: 'white',
    borderRadius: '0.5rem',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
    padding: '1.5rem'
  },
  userInfo: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: '1rem',
    flexWrap: 'wrap' as const,
    gap: '1rem'
  },
  inputGroup: {
    display: 'flex',
    flexDirection: 'column' as const
  },
  userNameRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem'
  },
  label: {
    fontSize: '0.875rem',
    fontWeight: '500',
    color: '#374151',
    marginBottom: '0.5rem'
  },
  voteStats: {
    textAlign: 'right' as const
  },
  voteStatsLabel: {
    fontSize: '0.875rem',
    color: '#6b7280'
  },
  voteStatsNumber: {
    fontSize: '1.5rem',
    fontWeight: 'bold',
    color: '#2563eb'
  },
  selectedCard: {
    textAlign: 'center' as const,
    marginTop: '1rem',
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    justifyContent: 'center'
  },
  selectedCardLabel: {
    fontSize: '0.875rem',
    color: '#6b7280',
    marginBottom: '0.5rem'
  },
  selectedCardDisplay: {
    width: '4rem',
    height: '5rem',
    borderRadius: '0.5rem',
    color: 'white',
    fontWeight: 'bold',
    fontSize: '1.25rem',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
  },
  sectionTitle: {
    fontSize: '1.125rem',
    fontWeight: '600',
    color: '#374151',
    marginBottom: '1rem'
  },
  cardsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(3rem, 1fr))',
    gap: '0.75rem',
    marginBottom: '1.5rem'
  },
  pokerCard: {
    width: '3rem',
    height: '4rem',
    borderRadius: '0.5rem',
    fontWeight: 'bold',
    color: 'white',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
    transition: 'all 0.2s',
    border: 'none',
    cursor: 'pointer',
    fontSize: '1rem'
  },
  pokerCardSelected: {
    transform: 'scale(1.05)',
    boxShadow: '0 0 0 4px rgba(59, 130, 246, 0.3)'
  },
  pokerCardDisabled: {
    cursor: 'not-allowed',
    opacity: 0.85,
    transform: 'none'
  },
  buttonsContainer: {
    display: 'flex',
    gap: '1rem',
    justifyContent: 'center',
    flexWrap: 'wrap' as const
  },
  button: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    padding: '0.75rem 1.5rem',
    borderRadius: '0.5rem',
    fontWeight: '600',
    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
    transition: 'background-color 0.2s',
    border: 'none',
    cursor: 'pointer',
    fontSize: '1rem'
  },
  primaryButton: {
    backgroundColor: '#3b82f6',
    color: 'white'
  },
  buttonDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed',
    transform: 'none'
  },
  warningButton: {
    backgroundColor: '#eab308',
    color: 'white'
  },
  participantsHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    marginBottom: '1rem'
  },
  participantsList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.75rem'
  },
  participant: {
    padding: '0.75rem',
    borderRadius: '0.5rem',
    border: '2px solid',
    transition: 'colors 0.2s'
  },
  participantSpectator: {
    borderColor: '#e5e7eb',
    backgroundColor: '#f9fafb'
  },
  participantVoted: {
    borderColor: '#bbf7d0',
    backgroundColor: '#f0fdf4'
  },
  participantWaiting: {
    borderColor: '#fef3c7',
    backgroundColor: '#fffbeb'
  },
  participantLowest: {
    borderColor: '#60a5fa',
    backgroundColor: '#eff6ff'
  },
  participantHighest: {
    borderColor: '#f97316',
    backgroundColor: '#fff7ed'
  },
  participantContent: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between'
  },
  participantName: {
    fontWeight: '500',
    color: '#1f2937',
    display: 'flex',
    alignItems: 'center',
    gap: '0.375rem'
  },
  participantStatus: {
    fontSize: '0.875rem',
    color: '#6b7280'
  },
  lowestVoteBadge: {
    padding: '0.125rem 0.375rem',
    borderRadius: '9999px',
    backgroundColor: '#dbeafe',
    color: '#1d4ed8',
    fontSize: '0.6875rem',
    fontWeight: '700'
  },
  highestVoteBadge: {
    padding: '0.125rem 0.375rem',
    borderRadius: '9999px',
    backgroundColor: '#ffedd5',
    color: '#c2410c',
    fontSize: '0.6875rem',
    fontWeight: '700'
  },
  participantRight: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem'
  },
  voteCard: {
    width: '2rem',
    height: '2.5rem',
    borderRadius: '0.25rem',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: 'white',
    fontWeight: 'bold',
    fontSize: '0.875rem'
  },
  statusDot: {
    width: '0.75rem',
    height: '0.75rem',
    borderRadius: '50%'
  },
  statusSpectator: {
    backgroundColor: '#9ca3af'
  },
  statusVoted: {
    backgroundColor: '#10b981'
  },
  statusWaiting: {
    backgroundColor: '#eab308'
  },
  summary: {
    marginTop: '1.5rem',
    padding: '1rem',
    backgroundColor: '#dbeafe',
    borderRadius: '0.5rem'
  },
  summaryTitle: {
    fontWeight: '600',
    color: '#1e40af',
    marginBottom: '0.5rem'
  },
  summaryContent: {
    fontSize: '0.875rem',
    color: '#1d4ed8'
  },
  adminControls: {
    display: 'flex',
    gap: '0.25rem'
  },
  roleButton: {
    width: '1.75rem',
    height: '1.75rem',
    borderRadius: '0.25rem',
    border: '1px solid #d1d5db',
    backgroundColor: 'white',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    transition: 'all 0.2s',
    fontSize: '0.75rem'
  },
  activeSpectatorButton: {
    backgroundColor: '#ddd6fe',
    borderColor: '#8b5cf6',
    color: '#6d28d9'
  },
  activeAdminButton: {
    backgroundColor: '#fef3c7',
    borderColor: '#f59e0b',
    color: '#92400e'
  },
  inactiveButton: {
    backgroundColor: 'white',
    borderColor: '#d1d5db',
    color: '#9ca3af'
  },
  roomHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '0.5rem 1.5rem',
    background: 'linear-gradient(135deg, #dbeafe 0%, #e0e7ff 100%)',
    boxShadow: 'none',
    marginBottom: '0'
  },
  roomInfo: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem'
  },
  roomLabel: {
    fontSize: '0.875rem',
    color: '#6b7280'
  },
  roomCode: {
    fontSize: '1rem',
    fontWeight: '600',
    color: '#1f2937',
    backgroundColor: '#e0e7ff',
    border: 'none',
    cursor: 'pointer',
    padding: '0.25rem 0.5rem',
    borderRadius: '0.25rem',
    fontFamily: 'Monaco, "Lucida Console", monospace'
  },
  backButton: {
    padding: '0.5rem 1rem',
    backgroundColor: '#f3f4f6',
    color: '#374151',
    border: '1px solid #d1d5db',
    borderRadius: '0.375rem',
    cursor: 'pointer',
    fontSize: '0.875rem',
    transition: 'background-color 0.2s'
  },
  disconnectedBanner: {
    backgroundColor: '#fef2f2',
    border: '1px solid #fca5a5',
    color: '#b91c1c',
    padding: '0.75rem 1rem',
    textAlign: 'center' as const,
    fontWeight: '500',
    borderRadius: '0.5rem',
    margin: '0.5rem',
  },

  backlogStoryPosition: {
    fontSize: '0.75rem',
    color: '#6b7280',
    marginLeft: '0.5rem',
    fontStyle: 'italic' as const,
    fontWeight: 400 as const
  },

};
