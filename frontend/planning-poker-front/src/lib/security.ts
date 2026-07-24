import type { LogEntry } from "./logger";

export type { LogEntry } from "./logger";

interface RateLimitEntry {
  count: number;
  windowStart: number;
}

const RATE_LIMIT_MAX = 100;
const RATE_LIMIT_WINDOW_MS = 60_000;
const SOURCE_REGEX = /^[a-zA-Z0-9._-]+$/;
const VALID_LEVELS = ["debug", "info", "warn", "error"] as const;

export class RateLimiter {
  private store = new Map<string, RateLimitEntry>();

  isAllowed(ip: string): boolean {
    const now = Date.now();
    this.pruneExpired(now);

    const entry = this.store.get(ip);
    if (!entry) {
      this.store.set(ip, { count: 1, windowStart: now });
      return true;
    }

    if (entry.count >= RATE_LIMIT_MAX) {
      return false;
    }

    entry.count += 1;
    return true;
  }

  private pruneExpired(now: number): void {
    for (const [ip, entry] of this.store) {
      if (now - entry.windowStart >= RATE_LIMIT_WINDOW_MS) {
        this.store.delete(ip);
      }
    }
  }
}

export function sanitizeString(str: string, maxLength: number): string {
  const truncated = str.length > maxLength ? str.slice(0, maxLength) : str;
  return truncated.replace(/[\x00-\x08\x0B-\x1F\x7F]/g, "");
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function sanitizeLogEntry(entry: unknown): LogEntry | null {
  if (!isPlainObject(entry)) {
    return null;
  }

  const { message, source, level, timestamp, sessionId, meta } = entry;

  if (typeof message !== "string") {
    return null;
  }

  if (typeof source !== "string" || !SOURCE_REGEX.test(source)) {
    return null;
  }

  if (typeof sessionId !== "string") {
    return null;
  }

  const sanitizedLevel =
    typeof level === "string" && (VALID_LEVELS as readonly string[]).includes(level)
      ? (level as LogEntry["level"])
      : "info";

  const sanitizedTimestamp =
    typeof timestamp === "string" ? timestamp : String(Date.now());

  const result: LogEntry = {
    message: sanitizeString(message, 1000),
    source: sanitizeString(source, 100),
    level: sanitizedLevel,
    timestamp: sanitizedTimestamp,
    sessionId,
  };

  if (meta !== undefined) {
    if (!isPlainObject(meta)) {
      return null;
    }

    const entries = Object.entries(meta);
    if (entries.length > 20) {
      return null;
    }

    const sanitizedMeta: Record<string, unknown> = {};
    for (const [key, val] of entries) {
      sanitizedMeta[sanitizeString(key, 50)] =
        typeof val === "string" ? sanitizeString(val, 500) : val;
    }
    result.meta = sanitizedMeta;
  }

  return result;
}

export function sanitizeLogPayload(body: unknown): LogEntry[] {
  if (!isPlainObject(body) || !Array.isArray(body.logs)) {
    return [];
  }

  const logs = body.logs as unknown[];
  if (logs.length > 50) {
    return [];
  }

  return logs
    .map((entry) => sanitizeLogEntry(entry))
    .filter((entry): entry is LogEntry => entry !== null);
}
