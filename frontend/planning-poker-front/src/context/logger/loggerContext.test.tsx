import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LogBus } from "@/lib/logger";
import { LoggerProvider, useLogger } from "./loggerContext";

function TestConsumer({
  onMethods,
}: {
  onMethods: (methods: ReturnType<typeof useLogger>) => void;
}) {
  const methods = useLogger("test-source");
  onMethods(methods);
  return <div>test</div>;
}

describe("LoggerProvider", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("starts LogBus on mount", () => {
    const startSpy = vi.spyOn(LogBus.prototype, "start");

    render(
      <LoggerProvider>
        <div>child</div>
      </LoggerProvider>,
    );

    expect(startSpy).toHaveBeenCalledOnce();
  });

  it("stops LogBus on unmount", () => {
    const stopSpy = vi.spyOn(LogBus.prototype, "stop");

    const { unmount } = render(
      <LoggerProvider>
        <div>child</div>
      </LoggerProvider>,
    );

    unmount();

    expect(stopSpy).toHaveBeenCalledOnce();
  });

  it("flushes LogBus on unmount", () => {
    const flushSpy = vi.spyOn(LogBus.prototype, "flush");

    const { unmount } = render(
      <LoggerProvider>
        <div>child</div>
      </LoggerProvider>,
    );

    unmount();

    expect(flushSpy).toHaveBeenCalled();
  });
});

describe("useLogger", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("returns logger methods bound to source", () => {
    let methods: ReturnType<typeof useLogger> | null = null;

    render(
      <LoggerProvider>
        <TestConsumer onMethods={(m) => { methods = m; }} />
      </LoggerProvider>,
    );

    expect(methods).not.toBeNull();
    expect(methods!.debug).toBeInstanceOf(Function);
    expect(methods!.info).toBeInstanceOf(Function);
    expect(methods!.warn).toBeInstanceOf(Function);
    expect(methods!.error).toBeInstanceOf(Function);
  });

  it("logs entries with the correct source", () => {
    const addEntrySpy = vi.spyOn(LogBus.prototype, "addEntry");
    let methods: ReturnType<typeof useLogger> | null = null;

    render(
      <LoggerProvider>
        <TestConsumer onMethods={(m) => { methods = m; }} />
      </LoggerProvider>,
    );

    act(() => {
      methods!.info("hello world");
    });

    expect(addEntrySpy).toHaveBeenCalledWith(
      expect.objectContaining({
        level: "info",
        message: "hello world",
        source: "test-source",
      }),
    );
  });

  it("includes meta when provided", () => {
    const addEntrySpy = vi.spyOn(LogBus.prototype, "addEntry");
    let methods: ReturnType<typeof useLogger> | null = null;

    render(
      <LoggerProvider>
        <TestConsumer onMethods={(m) => { methods = m; }} />
      </LoggerProvider>,
    );

    act(() => {
      methods!.warn("warning", { requestId: "123" });
    });

    expect(addEntrySpy).toHaveBeenCalledWith(
      expect.objectContaining({
        level: "warn",
        message: "warning",
        source: "test-source",
        meta: { requestId: "123" },
      }),
    );
  });

  it("returns no-op logger if used outside provider", () => {
    function BadConsumer() {
      const logger = useLogger("test");
      // Should not throw — returns a no-op logger
      logger.info("test");
      return <div data-testid="ok">ok</div>;
    }

    const { getByTestId } = render(<BadConsumer />);
    expect(getByTestId("ok")).toBeTruthy();
  });
});

describe("Global error listeners", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("captures unhandled errors and logs them", () => {
    const addEntrySpy = vi.spyOn(LogBus.prototype, "addEntry");

    render(
      <LoggerProvider>
        <div>child</div>
      </LoggerProvider>,
    );

    const error = new Error("test uncaught error");

    act(() => {
      window.dispatchEvent(
        new ErrorEvent("error", {
          message: "test uncaught error",
          error,
        }),
      );
    });

    expect(addEntrySpy).toHaveBeenCalledWith(
      expect.objectContaining({
        level: "error",
        source: "uncaught",
        message: "test uncaught error",
        meta: expect.objectContaining({
          stack: error.stack,
        }),
      }),
    );
  });

  it("captures unhandled promise rejections and logs them", () => {
    const addEntrySpy = vi.spyOn(LogBus.prototype, "addEntry");

    render(
      <LoggerProvider>
        <div>child</div>
      </LoggerProvider>,
    );

    const reason = new Error("rejected promise");
    const promise = Promise.reject(reason);
    promise.catch(() => {});

    act(() => {
      window.dispatchEvent(
        new PromiseRejectionEvent("unhandledrejection", {
          promise,
          reason,
        }),
      );
    });

    expect(addEntrySpy).toHaveBeenCalledWith(
      expect.objectContaining({
        level: "error",
        source: "uncaught",
        message: "rejected promise",
        meta: expect.objectContaining({
          stack: reason.stack,
        }),
      }),
    );
  });
});
