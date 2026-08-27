export const styles = {
  overlay: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    width: '100vw',
    height: '100vh',
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000
  },
  dialog: {
    backgroundColor: 'white',
    borderRadius: '0.5rem',
    padding: '2rem',
    maxWidth: '600px',
    width: '100%',
    maxHeight: '80vh',
    overflowY: 'auto' as const,
    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.15)'
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1rem'
  },
  dialogHeader: {
    fontSize: '1.125rem',
    fontWeight: 600,
    marginBottom: '0.75rem',
    color: '#1f2937'
  },
  closeButton: {
    background: 'none',
    border: 'none',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '0.25rem',
    color: '#6b7280'
  },
  button: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    padding: '0.75rem 1.5rem',
    borderRadius: '0.5rem',
    fontWeight: 600,
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
  sectionTitle: {
    fontSize: '1.125rem',
    fontWeight: 600,
    color: '#374151',
    marginBottom: '1rem'
  },
  backlogHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1rem'
  },
  backlogList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.5rem',
    marginBottom: '1rem'
  },
  backlogStory: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '0.75rem',
    borderRadius: '0.375rem',
    border: '2px solid',
    transition: 'colors 0.2s'
  },
  backlogStoryCurrent: {
    borderColor: '#3b82f6',
    backgroundColor: '#eff6ff'
  },
  backlogStoryVoted: {
    borderColor: '#bbf7d0',
    backgroundColor: '#f0fdf4'
  },
  backlogStoryPending: {
    borderColor: '#fef3c7',
    backgroundColor: '#fffbeb'
  },
  backlogStoryDragging: {
    opacity: 0.55
  },
  backlogStoryLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem'
  },
  backlogStoryIndex: {
    fontWeight: 600,
    color: '#6b7280',
    fontSize: '0.875rem',
    minWidth: '1.5rem'
  },
  backlogStoryName: {
    fontWeight: 500,
    color: '#1f2937'
  },
  backlogStoryTag: {
    fontSize: '0.75rem',
    padding: '0.125rem 0.5rem',
    borderRadius: '9999px',
    backgroundColor: '#3b82f6',
    color: 'white',
    fontWeight: '600'
  },
  backlogStoryTagVoted: {
    fontSize: '0.75rem',
    padding: '0.125rem 0.5rem',
    borderRadius: '9999px',
    backgroundColor: '#10b981',
    color: 'white',
    fontWeight: '600'
  },
  backlogStoryTagPending: {
    fontSize: '0.75rem',
    padding: '0.125rem 0.5rem',
    borderRadius: '9999px',
    backgroundColor: '#f59e0b',
    color: 'white',
    fontWeight: '600'
  },
  backlogStoryTagEstimated: {
    fontSize: '0.75rem',
    padding: '0.125rem 0.5rem',
    borderRadius: '9999px',
    backgroundColor: '#10b981',
    color: 'white',
    fontWeight: '600'
  },
  backlogStoryRight: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem'
  },
  backlogAddForm: {
    display: 'flex',
    gap: '0.5rem',
    marginBottom: '1rem'
  },
  backlogInput: {
    flex: 1,
    padding: '0.5rem',
    borderRadius: '0.375rem',
    border: '1px solid #d1d5db',
    fontSize: '0.875rem',
    outline: 'none'
  },
  backlogActions: {
    display: 'flex',
    gap: '0.5rem',
    justifyContent: 'flex-end'
  },
  dangerButton: {
    backgroundColor: '#ef4444',
    color: 'white'
  },
  dangerSmallButton: {
    backgroundColor: '#fecaca',
    color: '#dc2626',
    padding: '0.25rem 0.5rem',
    border: '1px solid #fca5a5'
  },
  primarySmallButton: {
    backgroundColor: '#dbeafe',
    color: '#1d4ed8',
    padding: '0.25rem 0.5rem',
    border: '1px solid #93c5fd',
    fontSize: '0.75rem'
  },
  secondarySmallButton: {
    backgroundColor: '#f3f4f6',
    color: '#374151',
    padding: '0.25rem 0.5rem',
    border: '1px solid #d1d5db',
    fontSize: '0.75rem'
  },
  confirmButtons: {
    display: 'flex',
    gap: '0.5rem',
    justifyContent: 'flex-end'
  }
};
