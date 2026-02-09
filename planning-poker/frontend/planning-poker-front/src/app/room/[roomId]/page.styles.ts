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
    marginBottom: '0.5rem'
  },
  storyText: {
    color: '#6b7280',
    fontStyle: 'italic'
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
  successButton: {
    backgroundColor: '#10b981',
    color: 'white'
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
  participantContent: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between'
  },
  participantName: {
    fontWeight: '500',
    color: '#1f2937'
  },
  participantStatus: {
    fontSize: '0.875rem',
    color: '#6b7280'
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
};
