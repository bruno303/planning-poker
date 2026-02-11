'use client'

import { CheckCircle, X, XCircle } from 'lucide-react';
import { useEffect, useState } from 'react';
import styles from './toastNotifications.module.css';

export type ToastVariant = 'error' | 'success';

export type Toast = {
  id: string;
  message: string;
  variant: ToastVariant;
};

type Props = {
  toasts: Toast[];
  onDismiss: (id: string) => void;
};

// Detect prefers-color-scheme to switch toast theme subtly
function usePrefersLight() {
  const [isLight, setIsLight] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: light)');
    const handler = (e: MediaQueryListEvent) => setIsLight(e.matches);
    setIsLight(mq.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);
  return isLight;
}

export default function ToastNotifications({ toasts, onDismiss }: Props) {
  const prefersLight = usePrefersLight();

  if (!toasts.length) return null;

  const iconFor = (variant: ToastVariant) => {
    if (variant === 'success') return <CheckCircle className={styles.iconSuccess} size={20} />;
    return <XCircle className={styles.iconError} size={20} />;
  };

  return (
    <div className={styles.container}>
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`${styles.toast} ${prefersLight ? styles.toastLight : ''}`}
        >
          {iconFor(toast.variant)}
          <div className={styles.message}>{toast.message}</div>
          <button
            type="button"
            aria-label="Dismiss notification"
            className={`${styles.close} ${prefersLight ? styles.closeLight : ''}`}
            onClick={() => onDismiss(toast.id)}
          >
            <X size={16} />
          </button>
        </div>
      ))}
    </div>
  );
}
