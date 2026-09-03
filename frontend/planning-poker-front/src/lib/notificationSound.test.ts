import { afterEach, describe, expect, it, vi } from 'vitest';
import { playNotificationSound } from './notificationSound';

describe('playNotificationSound', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    delete process.env.NEXT_PUBLIC_UNVOTED_NOTIFICATION_SOUND_ENABLED;
    delete process.env.NEXT_PUBLIC_UNVOTED_NOTIFICATION_SOUND_VOLUME;
  });

  it('plays a short sound using the configured volume', () => {
    process.env.NEXT_PUBLIC_UNVOTED_NOTIFICATION_SOUND_VOLUME = '0.4';
    const setValueAtTime = vi.fn();
    const linearRampToValueAtTime = vi.fn();
    const exponentialRampToValueAtTime = vi.fn();
    const oscillator = {
      type: '',
      frequency: { setValueAtTime: vi.fn(), linearRampToValueAtTime: vi.fn() },
      connect: vi.fn(),
      start: vi.fn(),
      stop: vi.fn(),
      addEventListener: vi.fn(),
    };
    const gain = { gain: { setValueAtTime, linearRampToValueAtTime, exponentialRampToValueAtTime }, connect: vi.fn() };
    const context = { currentTime: 10, destination: {}, createOscillator: vi.fn(() => oscillator), createGain: vi.fn(() => gain), resume: vi.fn().mockResolvedValue(undefined), close: vi.fn().mockResolvedValue(undefined) };
    vi.stubGlobal('AudioContext', vi.fn(() => context));

    playNotificationSound();

    expect(context.createOscillator).toHaveBeenCalledOnce();
    expect(linearRampToValueAtTime).toHaveBeenCalledWith(0.4, 10.03);
    expect(oscillator.stop).toHaveBeenCalledWith(10.8);
  });

  it('does nothing when sound is disabled', () => {
    process.env.NEXT_PUBLIC_UNVOTED_NOTIFICATION_SOUND_ENABLED = 'false';
    const audioContext = vi.fn();
    vi.stubGlobal('AudioContext', audioContext);

    playNotificationSound();

    expect(audioContext).not.toHaveBeenCalled();
  });

  it('ignores unavailable or failing audio APIs', () => {
    vi.stubGlobal('AudioContext', vi.fn(() => {
      throw new Error('autoplay blocked');
    }));

    expect(() => playNotificationSound()).not.toThrow();
  });

  it('ignores rejected resume and close operations', async () => {
    let ended: (() => void) | undefined;
    const context = {
      currentTime: 0,
      destination: {},
      createOscillator: vi.fn(() => ({
        type: '',
        frequency: { setValueAtTime: vi.fn(), linearRampToValueAtTime: vi.fn() },
        connect: vi.fn(),
        start: vi.fn(),
        stop: vi.fn(),
        addEventListener: vi.fn((_type: string, listener: () => void) => { ended = listener; }),
      })),
      createGain: vi.fn(() => ({
        gain: { setValueAtTime: vi.fn(), linearRampToValueAtTime: vi.fn(), exponentialRampToValueAtTime: vi.fn() },
        connect: vi.fn(),
      })),
      resume: vi.fn().mockRejectedValue(new Error('resume blocked')),
      close: vi.fn().mockRejectedValue(new Error('close blocked')),
    };
    vi.stubGlobal('AudioContext', vi.fn(() => context));

    expect(() => playNotificationSound()).not.toThrow();
    ended?.();
    await Promise.resolve();
    expect(context.close).toHaveBeenCalledOnce();
  });
});
