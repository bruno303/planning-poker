"use client";

import { LogBus, type LogContext, type LogEntry, type LogLevel } from "@/lib/logger";
import React, { createContext, useContext, useEffect, useMemo, useRef } from "react";

type LoggerContextType = {
  logger: LogBus;
  sessionId: string;
};

const LoggerContext = createContext<LoggerContextType | null>(null);

function generateSessionId(): string {
  try {
    return crypto.randomUUID();
  } catch {
    return `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
  }
}

export function LoggerProvider({ children }: { children: React.ReactNode }) {
  const loggerRef = useRef<LogBus | null>(null);
  const sessionIdRef = useRef(generateSessionId());

  if (!loggerRef.current) {
    loggerRef.current = new LogBus();
  }

  const logger = loggerRef.current;

  useEffect(() => {
    logger.start();

    const onError = (event: ErrorEvent) => {
      logger.addEntry({
        timestamp: new Date().toISOString(),
        level: "error",
        message: event.message || "Uncaught error",
        source: "uncaught",
        sessionId: sessionIdRef.current,
        meta: {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
          stack: event.error?.stack,
        },
      });
    };

    const onUnhandledRejection = (event: PromiseRejectionEvent) => {
      const reason = event.reason;
      logger.addEntry({
        timestamp: new Date().toISOString(),
        level: "error",
        message: reason instanceof Error ? reason.message : String(reason),
        source: "uncaught",
        sessionId: sessionIdRef.current,
        meta: {
          stack: reason instanceof Error ? reason.stack : undefined,
        },
      });
    };

    window.addEventListener("error", onError);
    window.addEventListener("unhandledrejection", onUnhandledRejection);

    return () => {
      window.removeEventListener("error", onError);
      window.removeEventListener("unhandledrejection", onUnhandledRejection);
      logger.stop();
    };
  }, [logger]);

  const value = useMemo(() => ({ logger, sessionId: sessionIdRef.current }), [logger]);

  return (
    <LoggerContext.Provider value={value}>
      {children}
    </LoggerContext.Provider>
  );
}

export function useLogger(source: string) {
  const context = useContext(LoggerContext);

  const noop = useMemo(
    () => ({
      debug: (_message: string, _meta?: Record<string, unknown>) => {},
      info: (_message: string, _meta?: Record<string, unknown>) => {},
      warn: (_message: string, _meta?: Record<string, unknown>) => {},
      error: (_message: string, _meta?: Record<string, unknown>) => {},
      setContext: (_context: LogContext) => {},
    }),
    [],
  );

  if (!context) {
    return noop;
  }

  const { logger, sessionId } = context;

  return useMemo(() => {
    function log(
      level: LogLevel,
      message: string,
      meta?: Record<string, unknown>,
    ): void {
      const entry: LogEntry = {
        timestamp: new Date().toISOString(),
        level,
        message,
        source,
        sessionId,
        meta,
      };
      logger.addEntry(entry);
    }

    return {
      debug: (message: string, meta?: Record<string, unknown>) =>
        log("debug", message, meta),
      info: (message: string, meta?: Record<string, unknown>) =>
        log("info", message, meta),
      warn: (message: string, meta?: Record<string, unknown>) =>
        log("warn", message, meta),
      error: (message: string, meta?: Record<string, unknown>) =>
        log("error", message, meta),
      setContext: (context: LogContext) => logger.setContext(context),
    };
  }, [logger, source, sessionId]);
}
