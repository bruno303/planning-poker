import { NextRequest } from "next/server";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

import { POST, config, resetRateLimiter } from "./route";

function makeRequest(
  body: unknown,
  headers: Record<string, string> = {},
): NextRequest {
  const defaultHeaders: Record<string, string> = {
    "content-type": "application/json",
  };

  return new NextRequest("http://localhost/api/logs", {
    method: "POST",
    headers: { ...defaultHeaders, ...headers },
    body: JSON.stringify(body),
  });
}

const validPayload = {
  logs: [
    {
      message: "test log",
      source: "web.client",
      level: "info",
      timestamp: "2024-01-01T00:00:00Z",
      sessionId: "abc-123",
    },
  ],
};

describe("POST /api/logs", () => {
  const originalEnv = { ...process.env };

  beforeEach(() => {
    resetRateLimiter();
    process.env.NEXT_PUBLIC_APP_URL = "http://localhost:3000";
    process.env.NEXT_PUBLIC_BACKEND_URL = "";
    process.env.LOKI_URL = "http://localhost:3100";
    process.env.LOG_ENV = "test";

    vi.restoreAllMocks();
  });

  afterEach(() => {
    process.env = { ...originalEnv };
    vi.restoreAllMocks();
  });

  describe("origin check", () => {
    it("returns 403 when origin is missing", async () => {
      const request = makeRequest(validPayload, {});
      const response = await POST(request);
      expect(response.status).toBe(403);
    });

    it("returns 403 when origin does not match", async () => {
      const request = makeRequest(validPayload, {
        origin: "http://evil.com",
      });
      const response = await POST(request);
      expect(response.status).toBe(403);
    });

    it("accepts matching origin", async () => {
      vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const request = makeRequest(validPayload, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(204);
    });

    it("accepts matching referer", async () => {
      vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const request = makeRequest(validPayload, {
        referer: "http://localhost:3000/rooms",
      });
      const response = await POST(request);
      expect(response.status).toBe(204);
    });
  });

  describe("rate limiting", () => {
    it("returns 429 when rate limit is exceeded", async () => {
      vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const headers = { origin: "http://localhost:3000" };

      // Exhaust rate limit (100 requests per window)
      for (let i = 0; i < 100; i++) {
        const req = makeRequest(validPayload, headers);
        await POST(req);
      }

      // 101st request should be rate limited
      const request = makeRequest(validPayload, headers);
      const response = await POST(request);
      expect(response.status).toBe(429);
    });
  });

  describe("payload validation", () => {
    it("returns 400 for invalid JSON", async () => {
      const request = new NextRequest("http://localhost/api/logs", {
        method: "POST",
        headers: {
          "content-type": "application/json",
          origin: "http://localhost:3000",
        },
        body: "not json",
      });
      const response = await POST(request);
      expect(response.status).toBe(400);
    });

    it("returns 400 for empty logs", async () => {
      const request = makeRequest({ logs: [] }, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(400);
    });

    it("returns 400 when all entries are invalid", async () => {
      const request = makeRequest(
        { logs: [null, "bad", { invalid: true }] },
        { origin: "http://localhost:3000" },
      );
      const response = await POST(request);
      expect(response.status).toBe(400);
    });
  });

  describe("sanitization", () => {
    it("filters out invalid entries and processes valid ones", async () => {
      vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const payload = {
        logs: [
          { message: "good", source: "web", sessionId: "a" },
          null,
          "bad",
          { message: "good2", source: "web", sessionId: "b" },
        ],
      };

      const request = makeRequest(payload, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(204);
    });
  });

  describe("missing LOKI_URL", () => {
    it("returns 503 when LOKI_URL is not set", async () => {
      delete process.env.LOKI_URL;

      const request = makeRequest(validPayload, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(503);
    });
  });

  describe("successful push", () => {
    it("returns 204 on successful Loki push", async () => {
      vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const request = makeRequest(validPayload, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(204);
    });

    it("returns 502 when Loki returns non-ok status", async () => {
      vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 500 }),
      );

      const request = makeRequest(validPayload, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(502);
    });

    it("returns 502 when fetch throws", async () => {
      vi.spyOn(global, "fetch").mockRejectedValue(new Error("network error"));

      const request = makeRequest(validPayload, {
        origin: "http://localhost:3000",
      });
      const response = await POST(request);
      expect(response.status).toBe(502);
    });
  });

  describe("Loki push payload format", () => {
    it("sends correctly shaped payload to Loki", async () => {
      const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const request = makeRequest(validPayload, {
        origin: "http://localhost:3000",
      });
      await POST(request);

      expect(fetchSpy).toHaveBeenCalledOnce();
      const [url, options] = fetchSpy.mock.calls[0]!;

      expect(url).toBe("http://localhost:3100/loki/api/v1/push");
      expect(options!.method).toBe("POST");

      const sentBody = JSON.parse(options!.body as string);
      expect(sentBody.streams).toHaveLength(1);

      const stream = sentBody.streams[0]!;
      expect(stream.stream).toEqual({
        service: "planning-poker-frontend",
        level: "info",
        env: "test",
        source: "web.client",
      });

      expect(stream.values).toHaveLength(1);
      const [timestamp, line] = stream.values[0]!;
      expect(timestamp).toMatch(/^\d+$/);
      expect(Number(timestamp)).toBeGreaterThan(1_000_000_000_000_000_000);

      const parsedLine = JSON.parse(line);
      expect(parsedLine.message).toBe("test log");
      expect(parsedLine.sessionId).toBe("abc-123");
    });

    it("groups entries with different levels into separate streams", async () => {
      const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const payload = {
        logs: [
          { message: "info msg", source: "web", sessionId: "a", level: "info" },
          { message: "error msg", source: "web", sessionId: "b", level: "error" },
          { message: "info msg 2", source: "web", sessionId: "c", level: "info" },
        ],
      };

      const request = makeRequest(payload, {
        origin: "http://localhost:3000",
      });
      await POST(request);

      const sentBody = JSON.parse(
        fetchSpy.mock.calls[0]![1]!.body as string,
      );
      expect(sentBody.streams).toHaveLength(2);

      const infoStream = sentBody.streams.find(
        (s: { stream: { level: string } }) => s.stream.level === "info",
      );
      const errorStream = sentBody.streams.find(
        (s: { stream: { level: string } }) => s.stream.level === "error",
      );

      expect(infoStream.values).toHaveLength(2);
      expect(errorStream.values).toHaveLength(1);
    });

    it("includes meta fields in log line", async () => {
      const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      const payload = {
        logs: [
          {
            message: "test",
            source: "web",
            sessionId: "a",
            meta: { component: "Header", action: "click" },
          },
        ],
      };

      const request = makeRequest(payload, {
        origin: "http://localhost:3000",
      });
      await POST(request);

      const sentBody = JSON.parse(
        fetchSpy.mock.calls[0]![1]!.body as string,
      );
      const line = JSON.parse(sentBody.streams[0].values[0][1]);
      expect(line.component).toBe("Header");
      expect(line.action).toBe("click");
    });
  });

  describe("body size config", () => {
    it("exports 64kb body size limit", () => {
      expect(config.api.bodyParser.sizeLimit).toBe("64kb");
    });
  });
});
