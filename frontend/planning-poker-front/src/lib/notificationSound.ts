const DEFAULT_VOLUME = 0.12;
const SOUND_DURATION_MS = 800;

function getNumberConfig(value: string | undefined, fallback: number): number {
  const parsedValue = Number.parseFloat(value ?? '');
  return Number.isFinite(parsedValue) ? parsedValue : fallback;
}

export function playNotificationSound(): void {
  if (process.env.NEXT_PUBLIC_UNVOTED_NOTIFICATION_SOUND_ENABLED === 'false' || typeof window === 'undefined') {
    return;
  }

  try {
    const AudioContextConstructor = window.AudioContext;
    if (!AudioContextConstructor) return;

    const context = new AudioContextConstructor();
    const oscillator = context.createOscillator();
    const gain = context.createGain();
    const now = context.currentTime;
    const volume = Math.min(1, Math.max(0, getNumberConfig(process.env.NEXT_PUBLIC_UNVOTED_NOTIFICATION_SOUND_VOLUME, DEFAULT_VOLUME)));

    oscillator.type = 'sine';
    oscillator.frequency.setValueAtTime(660, now);
    oscillator.frequency.linearRampToValueAtTime(880, now + 0.25);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(volume, now + 0.03);
    gain.gain.exponentialRampToValueAtTime(0.001, now + SOUND_DURATION_MS / 1000);
    oscillator.connect(gain);
    gain.connect(context.destination);
    oscillator.start(now);
    oscillator.stop(now + SOUND_DURATION_MS / 1000);
    oscillator.addEventListener('ended', () => {
      void context.close().catch(() => undefined);
    }, { once: true });
    void context.resume().catch(() => undefined);
  } catch {
    // Audio is an optional enhancement and may be blocked by the browser.
  }
}
