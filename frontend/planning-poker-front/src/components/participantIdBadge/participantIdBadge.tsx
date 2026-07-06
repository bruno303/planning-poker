'use client';

import { Hash } from 'lucide-react';
import { useState } from 'react';
import styles from './participantIdBadge.module.css';

type ParticipantIdBadgeProps = {
  participantId: string;
  onCopied: () => void;
};

export default function ParticipantIdBadge({ participantId, onCopied }: ParticipantIdBadgeProps) {
  const [showTooltip, setShowTooltip] = useState(false);

  const handleClick = async () => {
    try {
      await navigator.clipboard.writeText(participantId);
      onCopied();
    } catch (err) {
      // Clipboard API unavailable (non-HTTPS, Docker E2E, etc.) — fall back gracefully
      console.debug('Clipboard API unavailable', err);
    }
  };

  return (
    <span
      className={styles.container}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <button
        className={styles.badge}
        onClick={handleClick}
        title="Click to copy participant ID"
        type="button"
      >
        <Hash size={12} />
      </button>
      {showTooltip && (
        <div className={styles.tooltip}>
          <code>{participantId}</code>
          <span className={styles.hint}>Click to copy</span>
        </div>
      )}
    </span>
  );
}
