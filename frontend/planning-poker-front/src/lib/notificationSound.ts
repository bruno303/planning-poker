const NOTIFICATION_SOUND_VOLUME = 0.12;
const SOUND_DURATION_MS = 800;

export function playNotificationSound(): void {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    const AudioContextConstructor = window.AudioContext;
    if (!AudioContextConstructor) return;

    const context = new AudioContextConstructor();
    const oscillator = context.createOscillator();
    const gain = context.createGain();
    const now = context.currentTime;
    oscillator.type = 'sine';
    oscillator.frequency.setValueAtTime(660, now);
    oscillator.frequency.linearRampToValueAtTime(880, now + 0.25);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(NOTIFICATION_SOUND_VOLUME, now + 0.03);
    gain.gain.exponentialRampToValueAtTime(0.001, now + SOUND_DURATION_MS / 1000);
    oscillator.connect(gain);
    gain.connect(context.destination);
    oscillator.start(now);
    oscillator.stop(now + SOUND_DURATION_MS / 1000);
    let closed = false;
    let closeTimeout: ReturnType<typeof setTimeout> | undefined;
    const closeContext = () => {
      if (closed) return;
      closed = true;
      if (closeTimeout !== undefined) clearTimeout(closeTimeout);
      try {
        void context.close().catch(() => undefined);
      } catch {
        // Closing is best effort.
      }
    };
    closeTimeout = setTimeout(closeContext, SOUND_DURATION_MS + 100);
    oscillator.addEventListener('ended', closeContext, { once: true });
    void context.resume().catch(() => undefined);
  } catch {
    // Audio is an optional enhancement and may be blocked by the browser.
  }
}
