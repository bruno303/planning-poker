import { NextRequest, NextResponse } from "next/server";

import {
  LogEntry,
  RateLimiter,
  sanitizeLogPayload,
} from "@/lib/security";

let rateLimiter = new RateLimiter();

export function resetRateLimiter(): void {
  rateLimiter = new RateLimiter();
}

function getOrigin(request: NextRequest): string | null {
  const origin = request.headers.get("origin");
  if (origin) return origin;

  const referer = request.headers.get("referer");
  if (referer) {
    try {
      return new URL(referer).origin;
    } catch {
      return null;
    }
  }

  return null;
}

function getAllowedOrigins(): string[] {
  const origins: string[] = [];

  const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL;
  if (backendUrl) {
    try {
      origins.push(new URL(backendUrl).origin);
    } catch {
      // fall through
    }
  }

  const appUrl = process.env.NEXT_PUBLIC_APP_URL;
  if (appUrl) {
    try {
      origins.push(new URL(appUrl).origin);
    } catch {
      // fall through
    }
  }

  return origins;
}

function getClientIp(request: NextRequest): string {
  const forwarded = request.headers.get("x-forwarded-for");
  if (forwarded) {
    const firstIp = forwarded.split(",")[0]?.trim();
    if (firstIp) return firstIp;
  }
  return "unknown";
}

interface StreamGroup {
  service: string;
  level: string;
  env: string;
  source: string;
}

function buildLokiPayload(
  entries: LogEntry[],
  logEnv: string,
): {
  streams: Array<{
    stream: StreamGroup;
    values: Array<[string, string]>;
  }>;
} {
  const groups = new Map<string, Array<[string, string]>>();

  for (const entry of entries) {
    const key = JSON.stringify({
      service: "planning-poker-frontend",
      level: entry.level,
      env: logEnv,
      source: entry.source,
    });

    const line = JSON.stringify({
      message: entry.message,
      sessionId: entry.sessionId,
      ...(entry.meta ?? {}),
    });

    const dateMs = new Date(entry.timestamp).getTime();
    const timestamp = String(
      BigInt(Number.isFinite(dateMs) ? dateMs : Date.now()) * BigInt(1000000),
    );

    const existing = groups.get(key);
    if (existing) {
      existing.push([timestamp, line]);
    } else {
      groups.set(key, [[timestamp, line]]);
    }
  }

  const streams: Array<{
    stream: StreamGroup;
    values: Array<[string, string]>;
  }> = [];

  for (const [key, values] of groups) {
    const stream = JSON.parse(key) as StreamGroup;
    streams.push({ stream, values });
  }

  return { streams };
}

export async function POST(request: NextRequest) {
  // Origin check
  const allowedOrigins = getAllowedOrigins();
  if (allowedOrigins.length > 0) {
    const requestOrigin = getOrigin(request);
    if (!requestOrigin || !allowedOrigins.includes(requestOrigin)) {
      return NextResponse.json({ error: "Forbidden" }, { status: 403 });
    }
  }

  // Rate limiting
  const ip = getClientIp(request);
  if (!rateLimiter.isAllowed(ip)) {
    return NextResponse.json({ error: "Too Many Requests" }, { status: 429 });
  }

  // Parse JSON body
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON" }, { status: 400 });
  }

  // Sanitize payload
  const entries = sanitizeLogPayload(body);
  if (entries.length === 0) {
    return new NextResponse(null, { status: 204 });
  }

  // LOKI_URL check
  const lokiUrl = process.env.LOKI_URL;
  if (!lokiUrl) {
    return NextResponse.json(
      { error: "Loki not configured" },
      { status: 503 },
    );
  }

  const logEnv = process.env.LOG_ENV ?? "development";

  // Build and send payload
  const payload = buildLokiPayload(entries, logEnv);

  try {
    const response = await fetch(`${lokiUrl}/loki/api/v1/push`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: "Loki push failed" },
        { status: 502 },
      );
    }
  } catch {
    return NextResponse.json(
      { error: "Loki push failed" },
      { status: 502 },
    );
  }

  return new NextResponse(null, { status: 204 });
}

export const config = {
  api: {
    bodyParser: {
      sizeLimit: "64kb",
    },
  },
};
