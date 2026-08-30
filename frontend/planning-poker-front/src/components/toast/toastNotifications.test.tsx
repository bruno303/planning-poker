import { act } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ToastNotifications from './toastNotifications';
import styles from './toastNotifications.module.css';

const toasts = [
  { id: 'error', message: 'Failed', variant: 'error' as const },
  { id: 'success', message: 'Saved', variant: 'success' as const },
];

describe('ToastNotifications', () => {
  let change: (event: MediaQueryListEvent) => void;
  beforeEach(() => {
    change = () => {};
    vi.stubGlobal('matchMedia', vi.fn().mockImplementation(() => ({ matches: false, addEventListener: (_: string, fn: typeof change) => { change = fn; }, removeEventListener: vi.fn() })));
  });
  afterEach(() => { cleanup(); vi.restoreAllMocks(); });

  it('renders nothing for an empty list', () => {
    const { container } = render(<ToastNotifications toasts={[]} onDismiss={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders both variants and dismisses the selected toast', () => {
    const onDismiss = vi.fn();
    render(<ToastNotifications toasts={toasts} onDismiss={onDismiss} />);
    expect(screen.getByText('Failed')).toBeTruthy();
    expect(screen.getByText('Saved')).toBeTruthy();
    fireEvent.click(screen.getAllByRole('button', { name: 'Dismiss notification' })[1]);
    expect(onDismiss).toHaveBeenCalledWith('success');
  });

  it('applies the light theme and cleans up the media listener', async () => {
    const remove = vi.fn();
    const add = vi.fn();
    window.matchMedia = vi.fn().mockReturnValue({ matches: true, addEventListener: add, removeEventListener: remove }) as unknown as typeof window.matchMedia;
    const { unmount } = render(<ToastNotifications toasts={toasts} onDismiss={vi.fn()} />);
    await waitFor(() => expect(screen.getAllByText('Saved')[0].parentElement?.className.split(' ')).toContain(styles.toastLight));
    expect(add).toHaveBeenCalledWith('change', expect.any(Function));
    unmount();
    expect(remove).toHaveBeenCalledWith('change', add.mock.calls[0][1]);
  });
});
