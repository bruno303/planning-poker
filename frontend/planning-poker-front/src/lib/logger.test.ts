import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

beforeEach(() => {
  vi.stubEnv("NODE_ENV", "test");
  vi.stubEnv("NEXT_PUBLIC_LOG_LEVEL", "debug");
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
});

describe("createLogger", () => {
  it("respects min log level", async () => {
    vi.stubEnv("NEXT_PUBLIC_LOG_LEVEL", "warn");
    vi.resetModules();
    const { createLogger, getLogBus } = await import("./logger");

    const logger = createLogger("test");
    logger.debug("debug msg");
    logger.info("info msg");
    logger.warn("warn msg");
    logger.error("error msg");

    const bus = getLogBus();
    expect(bus.buffer).toHaveLength(2);
    expect(bus.buffer[0].level).toBe("warn");
    expect(bus.buffer[1].level).toBe("error");
  });

  it("includes sessionId", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "test-session-123" });
    vi.resetModules();
    const { createLogger, getLogBus } = await import("./logger");

    const logger = createLogger("test");
    logger.info("hello");

    expect(getLogBus().buffer[0].sessionId).toBe("test-session-123");
  });

  it("correct source", async () => {
    vi.resetModules();
    const { createLogger, getLogBus } = await import("./logger");

    const logger = createLogger("my.module");
    logger.info("hello");

    expect(getLogBus().buffer[0].source).toBe("my.module");
  });
});

describe("LogBus", () => {
  it("flush on threshold", async () => {
    vi.resetModules();
    const { createLogger, getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();
    bus.enabled = true;

    const logger = createLogger("test");
    for (let i = 0; i < 10; i++) {
      logger.info(`msg ${i}`);
    }

    await vi.advanceTimersByTimeAsync(0);
    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("flush on interval", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();
    bus.start();

    // Wait for health check to complete
    await vi.advanceTimersByTimeAsync(0);

    bus.addEntry({
      timestamp: new Date().toISOString(),
      level: "info",
      message: "test",
      source: "test",
      sessionId: "abc",
    });

    // Health check called fetch once; no flush yet
    expect(fetchSpy).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(5000);
    // Now fetch was called again for the flush
    expect(fetchSpy).toHaveBeenCalledTimes(2);

    bus.stop();
  });

  it("skip flush if in flight", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockReturnValue(new Promise(() => {}));
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();
    bus.enabled = true;

    for (let i = 0; i < 10; i++) {
      bus.addEntry({
        timestamp: new Date().toISOString(),
        level: "info",
        message: `msg ${i}`,
        source: "test",
        sessionId: "abc",
      });
    }

    expect(fetchSpy).toHaveBeenCalledTimes(1);

    for (let i = 0; i < 10; i++) {
      bus.addEntry({
        timestamp: new Date().toISOString(),
        level: "info",
        message: `msg ${i}`,
        source: "test",
        sessionId: "abc",
      });
    }

    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("buffer cap (200)", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockReturnValue(new Promise(() => {}));
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();

    for (let i = 0; i < 250; i++) {
      bus.addEntry({
        timestamp: new Date().toISOString(),
        level: "info",
        message: `msg ${i}`,
        source: "test",
        sessionId: "abc",
      });
    }

    expect(bus.buffer.length).toBeLessThanOrEqual(200);
  });

  it("circuit breaker on 503", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const fetchSpy = vi.fn().mockResolvedValue({ status: 503 });
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();
    bus.enabled = true;
    expect(bus.enabled).toBe(true);

    for (let i = 0; i < 10; i++) {
      bus.addEntry({
        timestamp: new Date().toISOString(),
        level: "info",
        message: `test ${i}`,
        source: "test",
        sessionId: "abc",
      });
    }

    await vi.advanceTimersByTimeAsync(0);

    expect(bus.enabled).toBe(false);
    expect(warnSpy).toHaveBeenCalledWith(
      "[Logger] Circuit breaker tripped: server returned 503",
    );

    for (let i = 0; i < 10; i++) {
      bus.addEntry({
        timestamp: new Date().toISOString(),
        level: "info",
        message: `test after ${i}`,
        source: "test",
        sessionId: "abc",
      });
    }

    await vi.advanceTimersByTimeAsync(0);

    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("sendBeacon on visibility hidden", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", fetchSpy);

    const sendBeaconSpy = vi.fn();
    vi.stubGlobal("navigator", { sendBeacon: sendBeaconSpy });

    const bus = getLogBus();
    bus.start();
    await vi.advanceTimersByTimeAsync(0);

    bus.addEntry({
      timestamp: new Date().toISOString(),
      level: "info",
      message: "test",
      source: "test",
      sessionId: "abc",
    });

    vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden");
    document.dispatchEvent(new Event("visibilitychange"));

    expect(sendBeaconSpy).toHaveBeenCalledTimes(1);
    expect(sendBeaconSpy).toHaveBeenCalledWith("/api/logs", expect.any(Blob));
  });
});

describe("flush payload format", () => {
  it("sends raw log entries to /api/logs", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();
    bus.enabled = true;

    bus.addEntry({
      timestamp: "2024-01-01T00:00:00.000Z",
      level: "info",
      message: "hello world",
      source: "test",
      sessionId: "abc",
      meta: { key: "value" },
    });

    bus.flush();
    await vi.advanceTimersByTimeAsync(0);

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body);
    expect(body.logs).toHaveLength(1);
    expect(body.logs[0].message).toBe("hello world");
    expect(body.logs[0].sessionId).toBe("abc");
    expect(body.logs[0].meta).toEqual({ key: "value" });
  });

  it("sends all entries as flat array", async () => {
    vi.resetModules();
    const { getLogBus } = await import("./logger");

    const fetchSpy = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", fetchSpy);

    const bus = getLogBus();
    bus.enabled = true;

    bus.addEntry({
      timestamp: new Date().toISOString(),
      level: "info",
      message: "msg 1",
      source: "service-a",
      sessionId: "abc",
    });
    bus.addEntry({
      timestamp: new Date().toISOString(),
      level: "error",
      message: "msg 2",
      source: "service-b",
      sessionId: "abc",
    });

    bus.flush();
    await vi.advanceTimersByTimeAsync(0);

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body);
    expect(body.logs).toHaveLength(2);
  });
});
