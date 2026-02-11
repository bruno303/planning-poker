'use client'

import ToastNotifications, { Toast, ToastVariant } from '@/components/toast/toastNotifications';
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';

type ToastContextType = {
  pushError: (message: string) => void;
  pushSuccess: (message: string) => void;
  dismiss: (id: string) => void;
};

const ToastContext = createContext<ToastContextType | null>(null);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timeouts = useRef<Record<string, number>>({});

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
    const timeoutId = timeouts.current[id];
    if (timeoutId !== undefined) {
      clearTimeout(timeoutId);
      delete timeouts.current[id];
    }
  }, []);

  const pushToast = useCallback((message: string, variant: ToastVariant) => {
    if (!message) return;
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    setToasts((prev) => [...prev, { id, message, variant }]);
    const timeoutId = window.setTimeout(() => dismiss(id), 10_000);
    timeouts.current[id] = timeoutId;
  }, [dismiss]);

  const pushError = useCallback((message: string) => pushToast(message, 'error'), [pushToast]);
  const pushSuccess = useCallback((message: string) => pushToast(message, 'success'), [pushToast]);

  useEffect(() => {
    return () => {
      Object.values(timeouts.current).forEach((timeoutId) => clearTimeout(timeoutId));
    };
  }, []);

  const value = useMemo(() => ({ pushError, pushSuccess, dismiss }), [pushError, pushSuccess, dismiss]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastNotifications toasts={toasts} onDismiss={dismiss} />
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return context;
}
