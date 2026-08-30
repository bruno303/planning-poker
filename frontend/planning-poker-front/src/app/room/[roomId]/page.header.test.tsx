import { act } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Header from './page.header';

const mockDebug = vi.fn();

vi.mock('@/context/logger/loggerContext', () => ({
  useLogger: () => ({ debug: mockDebug }),
}));

describe('Header', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn() },
    });
    mockDebug.mockClear();
    backHome.mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
    cleanup();
    vi.restoreAllMocks();
  });

  const renderHeader = () => render(
    <Header
      handleBackToHome={backHome}
      generateShareableLink={() => 'https://example.test/room/room-1'}
    >
      <p>Room content</p>
    </Header>,
  );

  const backHome = vi.fn();

  it('renders a native share button and navigates back home', () => {
    renderHeader();

    const shareButton = screen.getByRole('button', { name: 'Click here to share the room' });

    expect(shareButton).toBeInstanceOf(HTMLButtonElement);
    const back = screen.getByRole('button', { name: /back to home/i });
    fireEvent.click(back);
    expect(backHome).toHaveBeenCalledOnce();
  });

  it('does not show a success toast when clipboard writing is rejected', async () => {
    const writeText = navigator.clipboard.writeText as ReturnType<typeof vi.fn>;
    writeText.mockRejectedValue(new Error('Not allowed'));
    renderHeader();

    fireEvent.click(screen.getByRole('button', { name: 'Click here to share the room' }));

    await act(async () => {});
    expect(writeText).toHaveBeenCalledWith('https://example.test/room/room-1');
    expect(screen.queryByText('Shareable link copied!')).toBeNull();
    expect(mockDebug).toHaveBeenCalledWith('Clipboard API unavailable', { meta: { error: expect.stringContaining('Not allowed') } });
  });

  it('does not duplicate activation when Enter precedes the native click', async () => {
    const writeText = navigator.clipboard.writeText as ReturnType<typeof vi.fn>;
    writeText.mockResolvedValue(undefined);
    renderHeader();
    const shareButton = screen.getByRole('button', { name: 'Click here to share the room' });
    shareButton.focus();
    fireEvent.keyDown(shareButton, { key: 'Enter' });
    expect(writeText).not.toHaveBeenCalled();
    fireEvent.click(shareButton);
    await act(async () => {});
    expect(writeText).toHaveBeenCalledOnce();
    expect(writeText).toHaveBeenCalledWith('https://example.test/room/room-1');
    expect(screen.getByText('Shareable link copied!')).toBeTruthy();
  });
});
