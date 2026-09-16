'use client';

import BacklogModal from '@/components/backlogModal/backlogModal';
import FocusableComponent from '@/components/focusableInput/focusableInput';
import { RoomClock } from '@/components/roomClock/roomClock';
import { ChevronLeft, ChevronRight, List } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { Story } from '@/components/messages/websocket';
import { styles } from '../page.styles';

type Props = {
  currentStory: string;
  startedAt: string | null;
  backlogMode: boolean;
  stories: Story[];
  currentStoryIndex: number;
  snapshotIdentity: number;
  amIAdmin: boolean;
  onUpdateStory: (story: string) => void;
  onRemoveStory: (storyId: string) => void;
  onAddStory: (story: string) => void;
  onSelectStory: (storyId: string) => void;
  onReorderStory: (storyId: string, targetIndex: number) => void;
  onPreviousStory: () => void;
  onNextStory: () => void;
};

export default function RoomStoryCard({ currentStory, startedAt, backlogMode, stories, currentStoryIndex, snapshotIdentity, amIAdmin, onUpdateStory, onRemoveStory, onAddStory, onSelectStory, onReorderStory, onPreviousStory, onNextStory }: Props) {
  const [draft, setDraft] = useState(currentStory);
  const [editing, setEditing] = useState(false);
  const [showBacklog, setShowBacklog] = useState(false);
  const originalDraft = useRef('');

  useEffect(() => setDraft(currentStory), [snapshotIdentity]);

  const save = () => {
    const story = draft.trim();
    if (!story && backlogMode && stories.length > 0) {
      if (!window.confirm('The task name is empty. Do you want to remove this task?')) return;
      const current = stories[currentStoryIndex];
      if (current) onRemoveStory(current.id);
    } else {
      onUpdateStory(story);
    }
    setEditing(false);
  };

  const addStory = (story: string) => onAddStory(story.trim());

  return <>
    <div style={styles.storyCard}>
      <div style={styles.storyHeader}>
        <h2 style={styles.storyTitle}>Current Story</h2>
        <RoomClock startedAt={startedAt} style={styles.roomClock} />
        {amIAdmin && !editing && ((backlogMode && currentStory) || !backlogMode) && <button style={{ ...styles.button, ...styles.primaryButton, ...styles.storyEditButton }} onClick={() => { originalDraft.current = draft; setEditing(true); }}>Edit</button>}
      </div>
      <div style={styles.storyLine}>
        {amIAdmin ? <div style={styles.storyContent}>{editing ? <><FocusableComponent currentStory={draft} onChange={(event) => setDraft(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') save(); if (event.key === 'Escape') { setDraft(originalDraft.current); setEditing(false); } }} /><button style={{ ...styles.button, ...styles.primaryButton, padding: '0.5rem 1rem', fontSize: '0.875rem' }} onClick={save}>Save</button></> : <label style={{ ...styles.label, margin: 0, flex: 1, textAlign: 'center' }}>{draft}{backlogMode && stories.length > 0 && <span style={styles.backlogStoryPosition}>(Story {currentStoryIndex + 1} of {stories.length})</span>}</label>}</div> : <p style={{ ...styles.storyText, margin: 0, width: '100%', gridColumn: 2, textAlign: 'center' }}>{draft}</p>}
        <div style={styles.storyControls}>
           {backlogMode && amIAdmin && <div style={styles.storyNavigation}><button onClick={onPreviousStory} disabled={currentStoryIndex <= 0} aria-label="Previous Story" title="Previous Story" style={{ ...styles.storyControlButton, ...(currentStoryIndex <= 0 ? styles.buttonDisabled : {}) }}><ChevronLeft size={20} aria-hidden="true" /></button><button onClick={onNextStory} disabled={currentStoryIndex >= stories.length - 1} aria-label="Next Story" title="Next Story" style={{ ...styles.storyControlButton, ...(currentStoryIndex >= stories.length - 1 ? styles.buttonDisabled : {}) }}><ChevronRight size={20} aria-hidden="true" /></button></div>}
          {backlogMode && <button style={{ ...styles.button, ...styles.primaryButton, ...styles.storyBacklogButton }} aria-label="Open backlog" onClick={() => setShowBacklog(true)}><List size={18} aria-hidden="true" />Backlog</button>}
        </div>
      </div>
    </div>
     {showBacklog && <BacklogModal stories={stories} currentStoryIndex={currentStoryIndex} amIAdmin={amIAdmin} onClose={() => setShowBacklog(false)} onAddStory={addStory} onRemoveStory={onRemoveStory} onSelectStory={onSelectStory} onReorderStory={onReorderStory} />}
  </>;
}
