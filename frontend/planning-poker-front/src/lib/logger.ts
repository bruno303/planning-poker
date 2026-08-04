export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  meta?: Record<string, unknown>;
  source: string;
  sessionId: string;
}

export interface Logger {
  debug: (message: string, meta?: Record<string, unknown>) => void;
  info: (message: string, meta?: Record<string, unknown>) => void;
  warn: (message: string, meta?: Record<string, unknown>) => void;
  error: (message: string, meta?: Record<string, unknown>) => void;
}

const LEVEL_VALUES: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const BUFFER_MAX = 200;
const FLUSH_INTERVAL_MS = 5000;
const FLUSH_THRESHOLD = 10;

function generateSessionId(): string {
  try {
    return crypto.randomUUID();
  } catch {
    return `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
  }
}
const sessionId = generateSessionId();

export class LogBus {
  buffer: LogEntry[] = [];
  private inFlight = false;
  private _enabled = false;
  private intervalId: ReturnType<typeof setInterval> | null = null;
  private warnOnce = false;

  get enabled(): boolean {
    return this._enabled;
  }

  set enabled(value: boolean) {
    this._enabled = value;
  }

  addEntry(entry: LogEntry): void {
    if (this.buffer.length >= BUFFER_MAX) {
      this.buffer.shift();
    }
    this.buffer.push(entry);
    if (this.buffer.length >= FLUSH_THRESHOLD) {
      this.flush();
    }
  }

  flush(): void {
    if (!this._enabled || this.inFlight || this.buffer.length === 0) return;
    this.inFlight = true;

    const entries = this.buffer.splice(0);

    fetch("/api/logs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ logs: entries }),
    })
      .then((response) => {
        if (response.status === 503) {
          this._enabled = false;
          if (!this.warnOnce) {
            console.warn(
              "[Logger] Circuit breaker tripped: server returned 503",
            );
            this.warnOnce = true;
          }
        }
      })
      .catch(() => {})
      .finally(() => {
        this.inFlight = false;
      });
  }

  private flushWithBeacon(): void {
    if (!this._enabled || this.buffer.length === 0) return;

    const entries = this.buffer.splice(0);
    const blob = new Blob([JSON.stringify({ logs: entries })], {
      type: "application/json",
    });

    if (typeof navigator !== "undefined" && navigator.sendBeacon) {
      navigator.sendBeacon("/api/logs", blob);
    }
  }

  start(): void {
    fetch("/api/logs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ logs: [] }),
    })
      .then((response) => {
        if (response.ok) {
          this._enabled = true;
          this.intervalId = setInterval(() => this.flush(), FLUSH_INTERVAL_MS);
          if (typeof document !== "undefined") {
            document.addEventListener(
              "visibilitychange",
              this.onVisibilityChange,
            );
          }
        }
      })
      .catch(() => {});
  }

  stop(): void {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
    if (typeof document !== "undefined") {
      document.removeEventListener(
        "visibilitychange",
        this.onVisibilityChange,
      );
    }
    this.flush();
  }

  onVisibilityChange = (): void => {
    if (document.visibilityState === "hidden") {
      this.flushWithBeacon();
    }
  };
}

const logBus = new LogBus();

export function getLogBus(): LogBus {
  return logBus;
}

export function createLogger(source: string): Logger {
  const minLevel = getMinLevel();

  function log(
    level: LogLevel,
    message: string,
    meta?: Record<string, unknown>,
  ): void {
    if (LEVEL_VALUES[level] < minLevel) return;

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message,
      source,
      sessionId,
      meta,
    };

    logBus.addEntry(entry);
  }

  return {
    debug: (message, meta) => log("debug", message, meta),
    info: (message, meta) => log("info", message, meta),
    warn: (message, meta) => log("warn", message, meta),
    error: (message, meta) => log("error", message, meta),
  };
}

function getMinLevel(): number {
  const envLevel = process.env.NEXT_PUBLIC_LOG_LEVEL;
  if (envLevel && envLevel in LEVEL_VALUES) {
    return LEVEL_VALUES[envLevel as LogLevel];
  }
  return LEVEL_VALUES["debug"];
}
