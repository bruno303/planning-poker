import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Story } from '@/components/messages/websocket';
import BacklogModal from './backlogModal';

const CONFIRM_TEXT = 'This will remove the story backlog and keep only the current story. Are you sure?';

const renderBacklogModal = (overrides: Partial<{
  stories: Story[];
  currentStoryIndex: number;
  amIAdmin: boolean;
  onClose: ReturnType<typeof vi.fn>;
  onAddStory: ReturnType<typeof vi.fn>;
  onRemoveStory: ReturnType<typeof vi.fn>;
  onDisableBacklog: ReturnType<typeof vi.fn>;
}> = {}) => {
  const props = {
    stories: [] as Story[],
    currentStoryIndex: 0,
    amIAdmin: false,
    onClose: vi.fn(),
    onAddStory: vi.fn(),
    onRemoveStory: vi.fn(),
    onDisableBacklog: vi.fn(),
    ...overrides,
  };

  render(
    <BacklogModal
      stories={props.stories}
      currentStoryIndex={props.currentStoryIndex}
      amIAdmin={props.amIAdmin}
      onClose={props.onClose}
      onAddStory={props.onAddStory}
      onRemoveStory={props.onRemoveStory}
      onDisableBacklog={props.onDisableBacklog}
    />,
  );

  return props;
};

describe('BacklogModal', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('renders story list with status tags', () => {
    const stories: Story[] = [
      { name: 'Story one', mostAppearingVotes: [], voted: false },
      { name: 'Story two', result: 6.5, mostAppearingVotes: [6, 7], voted: true },
      { name: 'Story three', mostAppearingVotes: [], voted: false },
    ];

    renderBacklogModal({ stories, currentStoryIndex: 0 });

    expect(screen.getByText('Story one')).not.toBeNull();
    expect(screen.getByText('Story two')).not.toBeNull();
    expect(screen.getByText('Story three')).not.toBeNull();
    expect(screen.getByText('Current')).not.toBeNull();
    expect(screen.getByText('Avg: 6.5')).not.toBeNull();
  });

  it('renders admin controls when amIAdmin is true', () => {
    const stories: Story[] = [
      { name: 'Current story', mostAppearingVotes: [], voted: false },
      { name: 'Voted story', result: 3, mostAppearingVotes: [], voted: true },
      { name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];

    renderBacklogModal({ stories, currentStoryIndex: 0, amIAdmin: true });

    expect(screen.getByPlaceholderText('Enter story name...')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Add' })).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Disable Backlog' })).not.toBeNull();
    expect(screen.getAllByTitle('Remove story')).toHaveLength(1);
  });

  it('focuses the story input when opened by an administrator', () => {
    renderBacklogModal({ amIAdmin: true });

    expect(document.activeElement).toBe(screen.getByPlaceholderText('Enter story name...'));
  });

  it('hides admin controls when amIAdmin is false', () => {
    const stories: Story[] = [
      { name: 'Current story', mostAppearingVotes: [], voted: false },
      { name: 'Voted story', result: 3, mostAppearingVotes: [], voted: true },
      { name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];

    renderBacklogModal({ stories, currentStoryIndex: 0, amIAdmin: false });

    expect(screen.queryByPlaceholderText('Enter story name...')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Add' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Disable Backlog' })).toBeNull();
    expect(screen.queryAllByTitle('Remove story')).toHaveLength(0);
  });

  it('calls onAddStory with trimmed story on Add click and clears input', () => {
    const { onAddStory } = renderBacklogModal({ amIAdmin: true });

    const input = screen.getByPlaceholderText('Enter story name...');
    const addButton = screen.getByRole('button', { name: 'Add' });
    fireEvent.change(input, { target: { value: '  New Story  ' } });
    addButton.focus();
    fireEvent.click(addButton);

    expect(onAddStory).toHaveBeenCalledWith('New Story');
    expect((input as HTMLInputElement).value).toBe('');
    expect(document.activeElement).toBe(input);
  });

  it('calls onAddStory on Enter key', () => {
    const { onAddStory } = renderBacklogModal({ amIAdmin: true });

    const input = screen.getByPlaceholderText('Enter story name...');
    fireEvent.change(input, { target: { value: '  Story from Enter  ' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onAddStory).toHaveBeenCalledWith('Story from Enter');
  });

  it('does not call onAddStory for empty input on Add click', () => {
    const { onAddStory } = renderBacklogModal({ amIAdmin: true });

    const input = screen.getByPlaceholderText('Enter story name...');
    fireEvent.change(input, { target: { value: '   ' } });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    expect(onAddStory).not.toHaveBeenCalled();
  });

  it('calls onRemoveStory with correct index', () => {
    const stories: Story[] = [
      { name: 'Current story', mostAppearingVotes: [], voted: false },
      { name: 'Voted story', result: 3, mostAppearingVotes: [], voted: true },
      { name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];

    const { onRemoveStory } = renderBacklogModal({ stories, currentStoryIndex: 0, amIAdmin: true });

    fireEvent.click(screen.getAllByTitle('Remove story')[0]);

    expect(onRemoveStory).toHaveBeenCalledWith(2);
  });

  it('handles disable backlog confirm flow', () => {
    const { onDisableBacklog } = renderBacklogModal({ amIAdmin: true });

    fireEvent.click(screen.getByRole('button', { name: 'Disable Backlog' }));
    expect(screen.getByText(CONFIRM_TEXT)).not.toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByText(CONFIRM_TEXT)).toBeNull();
    expect(onDisableBacklog).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'Disable Backlog' }));
    fireEvent.click(screen.getByRole('button', { name: 'Disable' }));
    expect(onDisableBacklog).toHaveBeenCalledOnce();
  });

  it('closes via close button', () => {
    const { onClose } = renderBacklogModal();

    fireEvent.click(screen.getByRole('button', { name: 'Close' }));

    expect(onClose).toHaveBeenCalledOnce();
  });

  it('closes via Escape key', () => {
    const { onClose } = renderBacklogModal();

    fireEvent.keyDown(window, { key: 'Escape' });

    expect(onClose).toHaveBeenCalledOnce();
  });

  it('closes via backdrop click but not dialog click', () => {
    const { onClose } = renderBacklogModal();

    const dialog = screen.getByRole('dialog');
    fireEvent.click(dialog);
    expect(onClose).not.toHaveBeenCalled();

    const overlay = dialog.parentElement as HTMLElement;
    fireEvent.click(overlay);
    expect(onClose).toHaveBeenCalledOnce();
  });
});
