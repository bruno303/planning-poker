'use client'
import { Loader2 } from 'lucide-react';
import styles from './loadingSpinner.module.css';

export default function LoadingSpinner() {
  return (
    <div className={styles.container}>
      <Loader2 className={styles.spinner} size={48} />
    </div>
  );
}
