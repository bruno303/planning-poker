'use client';

import { useEffect, useRef, useState, type DragEvent, type MouseEvent } from 'react';
import { ChevronDown, ChevronUp, GripVertical, Plus, Trash2, X } from 'lucide-react';
import type { Story } from '@/components/messages/websocket';
import { styles } from './backlogModal.styles';

type BacklogModalProps = {
  stories: Story[];
  currentStoryIndex: number;
  backlogVersion: number;
  amIAdmin: boolean;
  onClose: () => void;
  onAddStory: (story: string) => void;
  onRemoveStory: (storyId: string, expectedBacklogVersion: number) => void;
  onSelectStory: (storyId: string) => void;
  onReorderStory: (storyId: string, targetIndex: number, expectedBacklogVersion: number) => void;
  onDisableBacklog: () => void;
};

export default function BacklogModal({
  stories,
  currentStoryIndex,
  backlogVersion,
  amIAdmin,
  onClose,
  onAddStory,
  onRemoveStory,
  onSelectStory,
  onReorderStory,
  onDisableBacklog,
}: BacklogModalProps) {
  const [newStoryInput, setNewStoryInput] = useState('');
  const [showConfirm, setShowConfirm] = useState(false);
  const [draggedStoryId, setDraggedStoryId] = useState<string | null>(null);
  const newStoryInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (amIAdmin) {
      newStoryInputRef.current?.focus();
    }
  }, [amIAdmin]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose]);

  const handleAddStory = () => {
    const trimmed = newStoryInput.trim();
    if (!trimmed) {
      return;
    }
    onAddStory(trimmed);
    setNewStoryInput('');
    newStoryInputRef.current?.focus();
  };

  const handleDisableBacklog = () => {
    setShowConfirm(false);
    onDisableBacklog();
  };

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  const handleDragStart = (event: DragEvent<HTMLDivElement>, storyId: string) => {
    if (!amIAdmin) {
      return;
    }
    setDraggedStoryId(storyId);
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', storyId);
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>, targetIndex: number) => {
    event.preventDefault();
    if (!amIAdmin) {
      return;
    }

    const storyId = event.dataTransfer.getData('text/plain') || draggedStoryId;
    if (storyId) {
      onReorderStory(storyId, targetIndex, backlogVersion);
    }
    setDraggedStoryId(null);
  };

  const handleMove = (storyId: string, targetIndex: number) => {
    onReorderStory(storyId, targetIndex, backlogVersion);
  };

  return (
    <div style={styles.overlay} onClick={handleOverlayClick}>
      <div style={styles.dialog} role="dialog" aria-label="Story Backlog">
        {showConfirm ? (
          <>
            <h3 style={styles.dialogHeader}>Disable Backlog Mode</h3>
            <p style={{ fontSize: '0.875rem', color: '#6b7280', marginBottom: '1.5rem' }}>
              This will remove the story backlog and keep only the current story. Are you sure?
            </p>
            <div style={styles.confirmButtons}>
              <button style={styles.button} onClick={() => setShowConfirm(false)}>
                Cancel
              </button>
              <button
                style={{ ...styles.button, ...styles.dangerButton }}
                onClick={handleDisableBacklog}
              >
                Disable
              </button>
            </div>
          </>
        ) : (
          <>
            <div style={styles.header}>
              <h2 style={styles.sectionTitle}>Story Backlog</h2>
              <div style={styles.backlogHeader}>
                {amIAdmin && (
                  <button
                    style={{ ...styles.button, ...styles.dangerButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }}
                    onClick={() => setShowConfirm(true)}
                  >
                    Disable Backlog
                  </button>
                )}
                <button style={styles.closeButton} aria-label="Close" onClick={onClose}>
                  <X size={18} />
                </button>
              </div>
            </div>

            <div style={styles.backlogList}>
              {stories.map((story, index) => (
                <div
                  key={story.id}
                  draggable={amIAdmin}
                  onDragStart={(event) => handleDragStart(event, story.id)}
                  onDragEnd={() => setDraggedStoryId(null)}
                  onDragOver={(event) => {
                    if (amIAdmin) {
                      event.preventDefault();
                      event.dataTransfer.dropEffect = 'move';
                    }
                  }}
                  onDrop={(event) => handleDrop(event, index)}
                  style={{
                    ...styles.backlogStory,
                    ...(story.voted ? styles.backlogStoryVoted : styles.backlogStoryPending),
                    ...(index === currentStoryIndex ? styles.backlogStoryCurrent : {}),
                    ...(draggedStoryId === story.id ? styles.backlogStoryDragging : {}),
                  }}
                >
                  <div style={styles.backlogStoryLeft}>
                    {amIAdmin && <GripVertical size={16} aria-label="Drag to reorder" />}
                    <span style={styles.backlogStoryIndex}>{index + 1}.</span>
                    <span
                      style={{
                        ...styles.backlogStoryName,
                        ...(story.voted ? { textDecoration: 'line-through', opacity: 0.7 } : {}),
                      }}
                    >
                      {story.name}
                    </span>
                    {index === currentStoryIndex && (
                      <span style={styles.backlogStoryTag}>Current</span>
                    )}
                    {!story.voted && index !== currentStoryIndex && (
                      <span style={styles.backlogStoryTagPending}>Pending</span>
                    )}
                    {story.voted && (
                      <span style={styles.backlogStoryTagEstimated}>Estimated</span>
                    )}
                    {story.voted && story.result != null && (
                      <span style={styles.backlogStoryTagVoted}>Avg: {story.result.toFixed(1)}</span>
                    )}
                  </div>
                  <div style={styles.backlogStoryRight}>
                    {amIAdmin && (
                      <>
                        {!story.voted && index !== currentStoryIndex && (
                          <button
                            style={{ ...styles.button, ...styles.primarySmallButton }}
                            onClick={() => onSelectStory(story.id)}
                            title="Select story"
                          >
                            Select
                          </button>
                        )}
                        <button
                          style={{ ...styles.button, ...styles.secondarySmallButton }}
                          onClick={() => handleMove(story.id, index - 1)}
                          disabled={index === 0}
                          aria-label={`Move ${story.name} up`}
                          title="Move up"
                        >
                          <ChevronUp size={14} />
                        </button>
                        <button
                          style={{ ...styles.button, ...styles.secondarySmallButton }}
                          onClick={() => handleMove(story.id, index + 1)}
                          disabled={index === stories.length - 1}
                          aria-label={`Move ${story.name} down`}
                          title="Move down"
                        >
                          <ChevronDown size={14} />
                        </button>
                        {!story.voted && index !== currentStoryIndex && (
                          <button
                            style={{ ...styles.button, ...styles.dangerSmallButton }}
                            onClick={() => onRemoveStory(story.id, backlogVersion)}
                            title="Remove story"
                          >
                            <Trash2 size={14} />
                          </button>
                        )}
                      </>
                    )}
                  </div>
                </div>
              ))}
            </div>

            {amIAdmin && (
              <div style={styles.backlogAddForm}>
                <input
                  type="text"
                  ref={newStoryInputRef}
                  value={newStoryInput}
                  onChange={(e) => setNewStoryInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleAddStory();
                    }
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
  );
}
