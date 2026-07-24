import { describe, expect, it, vi } from "vitest";

import {
  RateLimiter,
  sanitizeLogEntry,
  sanitizeLogPayload,
  sanitizeString,
} from "./security";

describe("RateLimiter", () => {
  it("allows requests under the limit", () => {
    const limiter = new RateLimiter();

    for (let i = 0; i < 99; i++) {
      expect(limiter.isAllowed("1.2.3.4")).toBe(true);
    }
  });

  it("blocks requests at the limit", () => {
    const limiter = new RateLimiter();

    for (let i = 0; i < 100; i++) {
      limiter.isAllowed("1.2.3.4");
    }

    expect(limiter.isAllowed("1.2.3.4")).toBe(false);
  });

  it("prunes expired entries so requests are allowed again", () => {
    const limiter = new RateLimiter();
    const now = Date.now();

    vi.spyOn(Date, "now").mockReturnValue(now);

    for (let i = 0; i < 100; i++) {
      limiter.isAllowed("1.2.3.4");
    }
    expect(limiter.isAllowed("1.2.3.4")).toBe(false);

    vi.mocked(Date.now).mockReturnValue(now + 61_000);
    expect(limiter.isAllowed("1.2.3.4")).toBe(true);

    vi.restoreAllMocks();
  });
});

describe("sanitizeString", () => {
  it("truncates to max length", () => {
    expect(sanitizeString("hello world", 5)).toBe("hello");
  });

  it("strips control characters except newline and tab", () => {
    expect(sanitizeString("a\x00b\x01c\x09d\x0Ae", 100)).toBe("abc\td\ne");
  });

  it("strips carriage return", () => {
    expect(sanitizeString("a\x0Db", 100)).toBe("ab");
  });

  it("preserves normal strings", () => {
    expect(sanitizeString("hello", 100)).toBe("hello");
  });
});

describe("sanitizeLogEntry", () => {
  const validEntry = {
    message: "test log",
    source: "web.client",
    level: "info",
    timestamp: "2024-01-01T00:00:00Z",
    sessionId: "abc-123",
  };

  it("passes valid entries", () => {
    const result = sanitizeLogEntry(validEntry);
    expect(result).toEqual({
      message: "test log",
      source: "web.client",
      level: "info",
      timestamp: "2024-01-01T00:00:00Z",
      sessionId: "abc-123",
    });
  });

  it("returns null for non-objects", () => {
    expect(sanitizeLogEntry(null)).toBeNull();
    expect(sanitizeLogEntry("string")).toBeNull();
    expect(sanitizeLogEntry(42)).toBeNull();
  });

  it("returns null for missing required fields", () => {
    expect(sanitizeLogEntry({ message: "test" })).toBeNull();
    expect(sanitizeLogEntry({ source: "web" })).toBeNull();
    expect(sanitizeLogEntry({ sessionId: "abc" })).toBeNull();
  });

  it("returns null for invalid source format", () => {
    expect(
      sanitizeLogEntry({ ...validEntry, source: "invalid source!" }),
    ).toBeNull();
  });

  it("defaults level to info for invalid values", () => {
    const result = sanitizeLogEntry({ ...validEntry, level: "invalid" });
    expect(result?.level).toBe("info");
  });

  it("defaults timestamp when not provided", () => {
    const before = Date.now();
    const result = sanitizeLogEntry({
      message: "test",
      source: "web",
      sessionId: "abc",
    });
    const ts = Number(result?.timestamp);
    expect(ts).toBeGreaterThanOrEqual(before);
    expect(ts).toBeLessThanOrEqual(Date.now());
  });

  it("truncates message to 1000 chars", () => {
    const longMessage = "a".repeat(1500);
    const result = sanitizeLogEntry({
      ...validEntry,
      message: longMessage,
    });
    expect(result?.message).toHaveLength(1000);
  });

  it("truncates source to 100 chars", () => {
    const longSource = "a".repeat(101);
    const result = sanitizeLogEntry({
      ...validEntry,
      source: longSource,
    });
    expect(result?.source).toHaveLength(100);
  });

  it("validates meta object constraints", () => {
    const validMeta = { key: "value" };
    const result = sanitizeLogEntry({ ...validEntry, meta: validMeta });
    expect(result?.meta).toEqual({ key: "value" });
  });

  it("preserves non-string meta values as-is", () => {
    const result1 = sanitizeLogEntry({ ...validEntry, meta: { key: 123 } });
    expect(result1?.meta).toEqual({ key: 123 });

    const result2 = sanitizeLogEntry({ ...validEntry, meta: { key: true } });
    expect(result2?.meta).toEqual({ key: true });

    const result3 = sanitizeLogEntry({ ...validEntry, meta: { key: null } });
    expect(result3?.meta).toEqual({ key: null });
  });

  it("returns null for meta with more than 20 keys", () => {
    const tooManyKeys: Record<string, string> = {};
    for (let i = 0; i < 21; i++) {
      tooManyKeys[`key${i}`] = "value";
    }
    expect(sanitizeLogEntry({ ...validEntry, meta: tooManyKeys })).toBeNull();
  });

  it("truncates meta keys to 50 chars", () => {
    const longKey = "a".repeat(51);
    const result = sanitizeLogEntry({
      ...validEntry,
      meta: { [longKey]: "value" },
    });
    const keys = Object.keys(result?.meta ?? {});
    expect(keys[0]).toHaveLength(50);
  });

  it("truncates meta values to 500 chars", () => {
    const longValue = "a".repeat(501);
    const result = sanitizeLogEntry({
      ...validEntry,
      meta: { key: longValue },
    });
    expect(result?.meta?.key).toHaveLength(500);
  });
});

describe("sanitizeLogPayload", () => {
  const validEntry = {
    message: "test",
    source: "web",
    sessionId: "abc",
  };

  it("returns sanitized entries for valid batch", () => {
    const result = sanitizeLogPayload({ logs: [validEntry, validEntry] });
    expect(result).toHaveLength(2);
  });

  it("filters out invalid entries", () => {
    const result = sanitizeLogPayload({
      logs: [validEntry, null, "bad", validEntry],
    });
    expect(result).toHaveLength(2);
  });

  it("returns empty array for oversized batch", () => {
    const logs = Array.from({ length: 51 }, () => validEntry);
    expect(sanitizeLogPayload({ logs })).toEqual([]);
  });

  it("returns empty array for missing logs field", () => {
    expect(sanitizeLogPayload({})).toEqual([]);
  });

  it("returns empty array for non-array logs", () => {
    expect(sanitizeLogPayload({ logs: "not-array" })).toEqual([]);
  });
});
