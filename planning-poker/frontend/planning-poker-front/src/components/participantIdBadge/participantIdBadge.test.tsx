import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import ParticipantIdBadge from './participantIdBadge';

describe('ParticipantIdBadge', () => {
  beforeEach(() => {
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn(),
      },
    });
    vi.spyOn(console, 'debug').mockImplementation(() => {});
  });
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  const TEST_UUID = '550e8400-e29b-41d4-a716-446655440000';
  const onCopied = vi.fn();

  it('calls onCopied when clipboard write succeeds', async () => {
    (navigator.clipboard.writeText as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

    render(
      <ParticipantIdBadge participantId={TEST_UUID} onCopied={onCopied} />,
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(TEST_UUID);
      expect(onCopied).toHaveBeenCalledOnce();
    });
  });

  it('does not call onCopied when clipboard write fails', async () => {
    (navigator.clipboard.writeText as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Not allowed'));

    render(
      <ParticipantIdBadge participantId={TEST_UUID} onCopied={onCopied} />,
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(TEST_UUID);
      expect(onCopied).not.toHaveBeenCalled();
      expect(console.debug).toHaveBeenCalledWith(
        'Clipboard API unavailable',
        expect.any(Error),
      );
    });
  });

  it('shows tooltip with UUID on mouse enter', () => {
    render(
      <ParticipantIdBadge participantId={TEST_UUID} onCopied={onCopied} />,
    );

    fireEvent.mouseEnter(screen.getByRole('button'));

    expect(screen.getByText(TEST_UUID)).not.toBeNull();
    expect(screen.getByText('Click to copy')).not.toBeNull();
  });

  it('hides tooltip on mouse leave', () => {
    render(
      <ParticipantIdBadge participantId={TEST_UUID} onCopied={onCopied} />,
    );

    fireEvent.mouseEnter(screen.getByRole('button'));
    fireEvent.mouseLeave(screen.getByRole('button'));

    expect(screen.queryByText(TEST_UUID)).toBeNull();
  });
});
