import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ComponentProps } from 'react';

import BacklogModal from './backlogModal';
import { Story } from '@/components/messages/websocket';

const makeProps = (overrides: Partial<ComponentProps<typeof BacklogModal>> = {}) => {
  const stories: Story[] = [
    { name: 'Story Alpha', mostAppearingVotes: [], voted: false },
    { name: 'Story Beta', mostAppearingVotes: [], voted: false },
    { name: 'Story Gamma', result: 6.5, mostAppearingVotes: [6.5], voted: true },
  ];
  return {
    stories,
    currentStoryIndex: 1,
    amIAdmin: true,
    onClose: vi.fn(),
    onAddStory: vi.fn(),
    onRemoveStory: vi.fn(),
    onDisableBacklog: vi.fn(),
    ...overrides,
  };
};

describe('BacklogModal', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('renders the story list with status tags', () => {
    render(<BacklogModal {...makeProps()} />);

    expect(screen.getByText('Story Alpha')).not.toBeNull();
    expect(screen.getByText('Story Beta')).not.toBeNull();
    expect(screen.getByText('Story Gamma')).not.toBeNull();
    expect(screen.getByText('Current')).not.toBeNull();
    expect(screen.getByText('Avg: 6.5')).not.toBeNull();
  });

  it('renders admin controls when amIAdmin is true', () => {
    render(<BacklogModal {...makeProps()} />);

    expect(screen.getByPlaceholderText('Enter story name...')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Add' })).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Disable Backlog' })).not.toBeNull();
    expect(screen.getAllByTitle('Remove story')).toHaveLength(1);
  });

  it('hides admin controls when amIAdmin is false', () => {
    render(<BacklogModal {...makeProps({ amIAdmin: false })} />);

    expect(screen.queryByPlaceholderText('Enter story name...')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Add' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Disable Backlog' })).toBeNull();
    expect(screen.queryAllByTitle('Remove story')).toHaveLength(0);
    expect(screen.getByText('Story Alpha')).not.toBeNull();
    expect(screen.getByText('Current')).not.toBeNull();
    expect(screen.getByText('Avg: 6.5')).not.toBeNull();
  });

  it('calls onAddStory with trimmed story on Add click and clears input', () => {
    const onAddStory = vi.fn();
    render(<BacklogModal {...makeProps({ onAddStory })} />);

    const input = screen.getByPlaceholderText('Enter story name...');
    fireEvent.change(input, { target: { value: '  New Story  ' } });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    expect(onAddStory).toHaveBeenCalledWith('New Story');
    expect((input as HTMLInputElement).value).toBe('');
  });

  it('calls onAddStory on Enter key', () => {
    const onAddStory = vi.fn();
    render(<BacklogModal {...makeProps({ onAddStory })} />);

    const input = screen.getByPlaceholderText('Enter story name...');
    fireEvent.change(input, { target: { value: '  New Story  ' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onAddStory).toHaveBeenCalledWith('New Story');
  });

  it('ignores empty input on Add click', () => {
    const onAddStory = vi.fn();
    render(<BacklogModal {...makeProps({ onAddStory })} />);

    fireEvent.change(screen.getByPlaceholderText('Enter story name...'), { target: { value: '   ' } });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    expect(onAddStory).not.toHaveBeenCalled();
  });

  it('calls onRemoveStory with the right index', () => {
    const onRemoveStory = vi.fn();
    render(<BacklogModal {...makeProps({ onRemoveStory })} />);

    fireEvent.click(screen.getByTitle('Remove story'));

    expect(onRemoveStory).toHaveBeenCalledWith(0);
  });

  it('supports the disable backlog confirm flow', () => {
    const onDisableBacklog = vi.fn();
    render(<BacklogModal {...makeProps({ onDisableBacklog })} />);

    fireEvent.click(screen.getByRole('button', { name: 'Disable Backlog' }));
    expect(
      screen.getByText('This will remove the story backlog and keep only the current story. Are you sure?'),
    ).not.toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onDisableBacklog).not.toHaveBeenCalled();
    expect(
      screen.queryByText('This will remove the story backlog and keep only the current story. Are you sure?'),
    ).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Disable Backlog' }));
    fireEvent.click(screen.getByRole('button', { name: 'Disable' }));

    expect(onDisableBacklog).toHaveBeenCalledOnce();
  });

  it('closes via the close button', () => {
    const onClose = vi.fn();
    render(<BacklogModal {...makeProps({ onClose })} />);

    fireEvent.click(screen.getByRole('button', { name: 'Close' }));

    expect(onClose).toHaveBeenCalledOnce();
  });

  it('closes via Escape key', () => {
    const onClose = vi.fn();
    render(<BacklogModal {...makeProps({ onClose })} />);

    fireEvent.keyDown(window, { key: 'Escape' });

    expect(onClose).toHaveBeenCalledOnce();
  });

  it('closes via backdrop click', () => {
    const onClose = vi.fn();
    render(<BacklogModal {...makeProps({ onClose })} />);

    const overlay = screen.getByRole('dialog').parentElement;
    expect(overlay).not.toBeNull();
    if (!overlay) return;

    fireEvent.click(overlay);

    expect(onClose).toHaveBeenCalledOnce();
  });
});