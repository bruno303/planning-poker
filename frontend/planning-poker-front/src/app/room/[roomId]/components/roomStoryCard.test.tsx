import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import RoomStoryCard from './roomStoryCard';
vi.mock('@/components/focusableInput/focusableInput', () => ({ default: (props: { currentStory: string; onChange: React.ChangeEventHandler<HTMLInputElement>; onKeyDown: React.KeyboardEventHandler<HTMLInputElement> }) => <input aria-label="Story editor" value={props.currentStory} onChange={props.onChange} onKeyDown={props.onKeyDown} /> }));

const props = (overrides = {}) => ({ currentStory: 'Original', startedAt: null, backlogMode: true, stories: [{ id: 'one', name: 'Original', mostAppearingVotes: [], voted: false }], currentStoryIndex: 0, snapshotIdentity: 1, amIAdmin: true, onUpdateStory: vi.fn(), onRemoveStory: vi.fn(), onAddStory: vi.fn(), onSelectStory: vi.fn(), onReorderStory: vi.fn(), onPreviousStory: vi.fn(), onNextStory: vi.fn(), ...overrides });

describe('RoomStoryCard', () => {
  afterEach(() => cleanup());
  it('overwrites drafts from authoritative snapshots and keeps cancel behavior', () => {
    const view = render(<RoomStoryCard {...props()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: 'local' } });
    view.rerender(<RoomStoryCard {...props({ snapshotIdentity: 2 })} />);
    expect((screen.getByLabelText('Story editor') as HTMLInputElement).value).toBe('Original');
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: 'cancelled' } });
    fireEvent.keyDown(screen.getByLabelText('Story editor'), { key: 'Escape' });
    expect(screen.getByText('Original')).toBeTruthy();
  });

  it('trims saved stories and confirms removal of an empty current story', () => {
    const onUpdateStory = vi.fn();
    const onRemoveStory = vi.fn();
    render(<RoomStoryCard {...props({ onUpdateStory, onRemoveStory })} />);
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: '  Renamed  ' } });
    fireEvent.keyDown(screen.getByLabelText('Story editor'), { key: 'Enter' });
    expect(onUpdateStory).toHaveBeenCalledWith('Renamed');
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Story editor'), { target: { value: ' ' } });
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(true));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onRemoveStory).toHaveBeenCalledWith('one');
  });
});
