import { act } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ToastProvider, useToast } from './toastContext';

type MockToast = {
  id: string;
  message: string;
  variant: 'error' | 'success';
};

const toastNotificationsSpy = vi.fn();

vi.mock('@/components/toast/toastNotifications', () => ({
  __esModule: true,
  default: ({ toasts, onDismiss }: { toasts: MockToast[]; onDismiss: (id: string) => void }) => {
    toastNotificationsSpy({ toasts, onDismiss });

    return (
      <div>
        {toasts.map((toast) => (
          <div key={toast.id}>
            <span>{toast.variant}</span>
            <span>{toast.message}</span>
            <button type="button" onClick={() => onDismiss(toast.id)}>
              dismiss-{toast.id}
            </button>
          </div>
        ))}
      </div>
    );
  },
}));

function TestConsumer() {
  const { pushError, pushSuccess } = useToast();

  return (
    <div>
      <button type="button" onClick={() => pushError('Something failed')}>
        push-error
      </button>
      <button type="button" onClick={() => pushSuccess('Saved successfully')}>
        push-success
      </button>
    </div>
  );
}

describe('ToastProvider', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    toastNotificationsSpy.mockClear();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
    cleanup();
  });

  it('keeps error toasts visible until dismissed manually', () => {
    render(
      <ToastProvider>
        <TestConsumer />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'push-error' }));

    expect(screen.getByText('Something failed')).not.toBeNull();

    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    expect(screen.getByText('Something failed')).not.toBeNull();

    const latestCall = toastNotificationsSpy.mock.lastCall?.[0] as {
      toasts: MockToast[];
      onDismiss: (id: string) => void;
    };

    fireEvent.click(screen.getByRole('button', { name: `dismiss-${latestCall.toasts[0].id}` }));

    expect(screen.queryByText('Something failed')).toBeNull();
  });

  it('auto-dismisses success toasts after ten seconds', () => {
    render(
      <ToastProvider>
        <TestConsumer />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'push-success' }));

    expect(screen.getByText('Saved successfully')).not.toBeNull();

    act(() => {
      vi.advanceTimersByTime(9_999);
    });
    expect(screen.getByText('Saved successfully')).not.toBeNull();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(screen.queryByText('Saved successfully')).toBeNull();
  });
});
