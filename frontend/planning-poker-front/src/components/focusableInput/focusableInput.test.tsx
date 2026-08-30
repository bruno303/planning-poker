import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import FocusableInput from './focusableInput';

describe('FocusableInput', () => {
  it('focuses on mount and forwards changes and key events', () => {
    const onChange = vi.fn();
    const onKeyDown = vi.fn();
    render(<FocusableInput currentStory="Story" onChange={onChange} onKeyDown={onKeyDown} />);
    const input = screen.getByDisplayValue('Story');
    expect(document.activeElement).toBe(input);
    fireEvent.change(input, { target: { value: 'Updated' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ type: 'change', target: input }));
    expect(onKeyDown).toHaveBeenCalledWith(expect.objectContaining({ key: 'Enter', target: input }));
  });
});
