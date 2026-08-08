'use client'

import { Story } from '@/components/messages/websocket'
import { Plus, Trash2, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { styles } from './backlogModal.styles'

type BacklogModalProps = {
  stories: Story[]
  currentStoryIndex: number
  amIAdmin: boolean
  onClose: () => void
  onAddStory: (story: string) => void
  onRemoveStory: (index: number) => void
  onDisableBacklog: () => void
}

function BacklogModal({
  stories,
  currentStoryIndex,
  amIAdmin,
  onClose,
  onAddStory,
  onRemoveStory,
  onDisableBacklog,
}: BacklogModalProps) {
  const [newStoryInput, setNewStoryInput] = useState('')
  const [showConfirm, setShowConfirm] = useState(false)

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const handleAddStory = () => {
    const trimmed = newStoryInput.trim()
    if (!trimmed) return
    onAddStory(trimmed)
    setNewStoryInput('')
  }

  return (
    <div
      style={styles.overlay}
      onClick={event => {
        if (event.target === event.currentTarget) {
          onClose()
        }
      }}
    >
      <div style={styles.dialog} role="dialog" aria-label="Story Backlog">
        {showConfirm ? (
          <div>
            <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.75rem', color: '#1f2937' }}>
              Disable Backlog Mode
            </h3>
            <p style={{ fontSize: '0.875rem', color: '#6b7280', marginBottom: '1.5rem' }}>
              This will remove the story backlog and keep only the current story. Are you sure?
            </p>
            <div style={styles.confirmButtons}>
              <button
                style={{ ...styles.button, background: '#e5e7eb', color: '#374151' }}
                onClick={() => setShowConfirm(false)}
              >
                Cancel
              </button>
              <button
                style={{ ...styles.button, ...styles.dangerButton }}
                onClick={() => {
                  setShowConfirm(false)
                  onDisableBacklog()
                }}
              >
                Disable
              </button>
            </div>
          </div>
        ) : (
          <>
            <div style={styles.header}>
              <h2 style={styles.sectionTitle}>Story Backlog</h2>
              <div style={styles.dialogHeader}>
                {amIAdmin && (
                  <button
                    style={{ ...styles.button, ...styles.dangerButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                    onClick={() => setShowConfirm(true)}
                  >
                    Disable Backlog
                  </button>
                )}
                <button style={styles.closeButton} onClick={onClose} aria-label="Close">
                  <X size={20} />
                </button>
              </div>
            </div>

            {stories.length > 0 && (
              <div style={styles.backlogList}>
                {stories.map((story, index) => (
                  <div
                    key={index}
                    style={{
                      ...styles.backlogStory,
                      ...(index === currentStoryIndex ? styles.backlogStoryCurrent : {}),
                      ...(story.voted ? styles.backlogStoryVoted : styles.backlogStoryPending),
                    }}
                  >
                    <div style={styles.backlogStoryLeft}>
                      <span style={styles.backlogStoryIndex}>{index + 1}.</span>
                      <span
                        style={{
                          ...styles.backlogStoryName,
                          ...(story.voted ? { textDecoration: 'line-through', opacity: 0.7 } : {}),
                        }}
                      >
                        {story.name}
                      </span>
                      {index === currentStoryIndex && !story.voted && (
                        <span style={styles.backlogStoryTag}>Current</span>
                      )}
                      {story.voted && story.result != null && (
                        <span style={styles.backlogStoryTagVoted}>
                          Avg: {story.result.toFixed(1)}
                        </span>
                      )}
                    </div>
                    <div style={styles.backlogStoryRight}>
                      {amIAdmin && !story.voted && index !== currentStoryIndex && (
                        <button
                          style={{ ...styles.button, ...styles.dangerSmallButton }}
                          onClick={() => onRemoveStory(index)}
                          title="Remove story"
                        >
                          <Trash2 size={14} />
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}

            {amIAdmin && (
              <div style={styles.backlogAddForm}>
                <input
                  type="text"
                  value={newStoryInput}
                  onChange={e => setNewStoryInput(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter') handleAddStory()
                  }}
                  placeholder="Enter story name..."
                  style={styles.backlogInput}
                />
                <button
                  style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                  onClick={handleAddStory}
                >
                  <Plus size={16} />
                  Add
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

export default BacklogModal