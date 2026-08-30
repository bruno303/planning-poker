import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Story } from '@/components/messages/websocket';
import BacklogModal from './backlogModal';

const renderBacklogModal = (overrides: Partial<{
  stories: Story[];
  currentStoryIndex: number;
  amIAdmin: boolean;
  onClose: ReturnType<typeof vi.fn>;
  onAddStory: ReturnType<typeof vi.fn>;
  onRemoveStory: ReturnType<typeof vi.fn>;
  onSelectStory: ReturnType<typeof vi.fn>;
  onReorderStory: ReturnType<typeof vi.fn>;
}> = {}) => {
  const props = {
    stories: [] as Story[],
    currentStoryIndex: 0,
    amIAdmin: false,
    onClose: vi.fn(),
    onAddStory: vi.fn(),
    onRemoveStory: vi.fn(),
    onSelectStory: vi.fn(),
    onReorderStory: vi.fn(),
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
      onSelectStory={props.onSelectStory}
      onReorderStory={props.onReorderStory}
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
      { id: 'story-1', name: 'Story one', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Story two', result: 6.5, mostAppearingVotes: [6, 7], voted: true },
      { id: 'story-3', name: 'Story three', mostAppearingVotes: [], voted: false },
    ];

    renderBacklogModal({ stories, currentStoryIndex: 0 });

    expect(screen.getByText('Story one')).not.toBeNull();
    expect(screen.getByText('Story two')).not.toBeNull();
    expect(screen.getByText('Story three')).not.toBeNull();
    expect(screen.getByText('Current')).not.toBeNull();
    expect(screen.getByText('Pending')).not.toBeNull();
    expect(screen.getByText('Estimated')).not.toBeNull();
    expect(screen.getByText('Avg: 6.5')).not.toBeNull();
  });

  it('renders the backlog with native dialog semantics and resets native defaults', () => {
    renderBacklogModal();

    const dialog = screen.getByRole('dialog', { name: 'Story Backlog' });
    expect(dialog.tagName).toBe('DIALOG');
    expect(dialog.hasAttribute('open')).toBe(true);
    expect((dialog as HTMLElement).style.position).toBe('relative');
    expect((dialog as HTMLElement).style.left).toBe('auto');
    expect((dialog as HTMLElement).style.right).toBe('auto');
  });

  it('renders admin controls when amIAdmin is true', () => {
    const stories: Story[] = [
      { id: 'story-1', name: 'Current story', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Voted story', result: 3, mostAppearingVotes: [], voted: true },
      { id: 'story-3', name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];

    renderBacklogModal({ stories, currentStoryIndex: 0, amIAdmin: true });

    expect(screen.getByPlaceholderText('Enter story name...')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Add' })).not.toBeNull();
    expect(screen.queryByRole('button', { name: 'Disable Backlog' })).toBeNull();
    expect(screen.getAllByTitle('Remove story')).toHaveLength(1);
    expect(screen.getAllByTitle('Select story')).toHaveLength(1);
    expect(screen.getAllByTitle('Move up')).toHaveLength(3);
    expect(screen.getAllByTitle('Move down')).toHaveLength(3);
  });

  it('focuses the story input when opened by an administrator', () => {
    renderBacklogModal({ amIAdmin: true });

    expect(document.activeElement).toBe(screen.getByPlaceholderText('Enter story name...'));
  });

  it('hides admin controls when amIAdmin is false', () => {
    const stories: Story[] = [
      { id: 'story-1', name: 'Current story', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Voted story', result: 3, mostAppearingVotes: [], voted: true },
      { id: 'story-3', name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];

    renderBacklogModal({ stories, currentStoryIndex: 0, amIAdmin: false });

    expect(screen.queryByPlaceholderText('Enter story name...')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Add' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Disable Backlog' })).toBeNull();
    expect(screen.queryAllByTitle('Remove story')).toHaveLength(0);
    expect(screen.queryAllByTitle('Select story')).toHaveLength(0);
    expect(screen.queryAllByTitle('Move up')).toHaveLength(0);
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

  it('calls onRemoveStory with stable id', () => {
    const stories: Story[] = [
      { id: 'story-1', name: 'Current story', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Voted story', result: 3, mostAppearingVotes: [], voted: true },
      { id: 'story-3', name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];

    const { onRemoveStory } = renderBacklogModal({ stories, currentStoryIndex: 0, amIAdmin: true });

    fireEvent.click(screen.getAllByTitle('Remove story')[0]);

    expect(onRemoveStory).toHaveBeenCalledWith('story-3');
  });

  it('selects pending stories by stable id', () => {
    const stories: Story[] = [
      { id: 'story-1', name: 'Current story', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];
    const { onSelectStory } = renderBacklogModal({ stories, amIAdmin: true });

    fireEvent.click(screen.getByTitle('Select story'));

    expect(onSelectStory).toHaveBeenCalledWith('story-2');
  });

  it('sends move button commands with stable id and target index', () => {
    const stories: Story[] = [
      { id: 'story-1', name: 'Current story', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];
    const { onReorderStory } = renderBacklogModal({ stories, amIAdmin: true });

    fireEvent.click(screen.getByRole('button', { name: 'Move Pending story up' }));

    expect(onReorderStory).toHaveBeenCalledWith('story-2', 0);
  });

  it('supports native drag and drop with stable id and target index', () => {
    const stories: Story[] = [
      { id: 'story-1', name: 'Current story', mostAppearingVotes: [], voted: false },
      { id: 'story-2', name: 'Pending story', mostAppearingVotes: [], voted: false },
    ];
    const { onReorderStory } = renderBacklogModal({ stories, amIAdmin: true });
    const currentRow = screen.getByText('Current story').parentElement?.parentElement as HTMLElement;
    const pendingRow = screen.getByText('Pending story').parentElement?.parentElement as HTMLElement;
    const dataTransfer = {
      effectAllowed: '',
      dropEffect: '',
      setData: vi.fn(),
      getData: vi.fn().mockReturnValue('story-2'),
    };

    fireEvent.dragStart(pendingRow, { dataTransfer });
    fireEvent.drop(currentRow, { dataTransfer });

    expect(onReorderStory).toHaveBeenCalledWith('story-2', 0);
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
