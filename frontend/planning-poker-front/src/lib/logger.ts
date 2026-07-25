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

const sessionId = crypto.randomUUID();

interface LokiStream {
  stream: Record<string, string>;
  values: Array<[string, string]>;
}

interface LokiPayload {
  streams: LokiStream[];
}

export class LogBus {
  buffer: LogEntry[] = [];
  private inFlight = false;
  private _enabled = process.env.NEXT_PUBLIC_LOG_ENABLED !== "false";
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
    const payload = this.buildPayload(entries);

    fetch("/api/logs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
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

  buildPayload(entries: LogEntry[]): LokiPayload {
    const env = process.env.NODE_ENV || "development";
    const groups = new Map<string, LogEntry[]>();

    for (const entry of entries) {
      const key = `${entry.source}|${entry.level}|${env}`;
      if (!groups.has(key)) {
        groups.set(key, []);
      }
      groups.get(key)!.push(entry);
    }

    return {
      streams: Array.from(groups.entries()).map(([key, groupEntries]) => {
        const [source, level, groupEnv] = key.split("|");
        return {
          stream: { service: source, level, env: groupEnv, source },
          values: groupEntries.map((e) => [
            String(BigInt(new Date(e.timestamp).getTime()) * BigInt(1000000)),
            JSON.stringify({
              message: e.message,
              meta: e.meta,
              sessionId: e.sessionId,
            }),
          ]),
        };
      }),
    };
  }

  private flushWithBeacon(): void {
    if (!this._enabled || this.buffer.length === 0) return;

    const entries = this.buffer.splice(0);
    const payload = this.buildPayload(entries);
    const blob = new Blob([JSON.stringify(payload)], {
      type: "application/json",
    });

    if (typeof navigator !== "undefined" && navigator.sendBeacon) {
      navigator.sendBeacon("/api/logs", blob);
    }
  }

  start(): void {
    this.intervalId = setInterval(() => this.flush(), FLUSH_INTERVAL_MS);
    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", this.onVisibilityChange);
    }
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
