import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import Avatar from './avatar';

const firstParticipant = { id: '550e8400-e29b-41d4-a716-446655440000' };
const validAvatarSources = [
  '/avatars/fox.svg',
  '/avatars/dog.svg',
  '/avatars/cat.svg',
  '/avatars/hammer.svg',
  '/avatars/strawberry.svg',
  '/avatars/wrench.svg',
  '/avatars/apple.svg',
  '/avatars/screwdriver.svg',
  '/avatars/lemon.svg',
];
const validAvatarLabels = [
  'Fox avatar',
  'Dog avatar',
  'Cat avatar',
  'Hammer avatar',
  'Strawberry avatar',
  'Wrench avatar',
  'Apple avatar',
  'Screwdriver avatar',
  'Lemon avatar',
];

describe('Avatar', () => {
  afterEach(() => {
    cleanup();
  });

  it('selects the same catalog asset for the same participant ID', () => {
    const { unmount } = render(<Avatar participant={firstParticipant} />);
    const firstSrc = screen.getByRole('img').getAttribute('src');
    unmount();

    render(<Avatar participant={firstParticipant} />);

    expect(screen.getByRole('img').getAttribute('src')).toBe(firstSrc);
  });

  it('selects valid catalog entries for different participant IDs', () => {
    const participantIds = [
      '550e8400-e29b-41d4-a716-446655440000',
      '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
      '6ba7b811-9dad-21d1-90b4-00c04fd430c8',
    ];

    const sources = participantIds.map((id) => {
      const { unmount } = render(<Avatar participant={{ id }} />);
      const source = screen.getByRole('img').getAttribute('src');
      unmount();
      return source;
    });

    expect(sources.every((source) => validAvatarSources.includes(source ?? ''))).toBe(true);
  });

  it.each(['', 'not-a-uuid', '  not-a-uuid  ', '550e8400-e29b-41d4-a716-44665544000z'])(
    'uses the fallback for malformed participant ID %j',
    (id) => {
      render(<Avatar participant={{ id }} />);

      const image = screen.getByRole('img');

      expect(image.getAttribute('src')).toBe('/avatars/fox.svg');
      expect(image.getAttribute('alt')).toBe('Fox avatar');
    },
  );

  it('renders an image with the selected avatar accessible label', () => {
    render(<Avatar participant={firstParticipant} />);

    const image = screen.getByRole('img');

    expect(image.getAttribute('alt')).toBeTruthy();
    expect(validAvatarLabels).toContain(image.getAttribute('alt'));
  });

  it('renders the avatar at the expected small dimensions', () => {
    render(<Avatar participant={firstParticipant} />);

    const image = screen.getByRole('img');

    expect(image.getAttribute('width')).toBe('32');
    expect(image.getAttribute('height')).toBe('32');
  });

  it('keeps the selected asset stable across rerenders', () => {
    const { rerender } = render(<Avatar participant={firstParticipant} />);
    const firstSrc = screen.getByRole('img').getAttribute('src');

    rerender(<Avatar participant={firstParticipant} />);

    expect(screen.getByRole('img').getAttribute('src')).toBe(firstSrc);
  });
});
