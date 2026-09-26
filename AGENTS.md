# AGENTS.md

Repository guide for coding agents working in `planning-poker`.

## Project Overview

Full-stack Planning Poker app for real-time agile estimation. A Go HTTP + WebSocket API (`cmd/api`) owns room state, backed by Redis for persistence, pub/sub broadcast, and locking. A Next.js 15 app (`frontend/planning-poker-front`) provides the UI and talks to the backend directly over REST and WebSocket. Local infra runs in Docker Compose; Kubernetes deployment uses the `helm/` chart.

API surface: `/health`, `/planning/rooms`, `/planning/room/{roomID}`, `/planning/{roomID}/ws`, admin routes under `/admin/rooms/...` (Bearer `ADMIN_API_KEY`), Swagger at `/swagger/{rest:.*}` when `ENVIRONMENT != production`, Prometheus metrics at `:9090/metrics`.

## Stack Summary

| Area | Evidence |
|---|---|
| Backend | Go 1.25.4, module `planning-poker` (`go.mod`); gorilla/mux, gorilla/websocket, go-redis v9, OpenTelemetry, `github.com/bruno303/go-toolkit` |
| Backend tooling | golangci-lint v2 (`.golangci.yaml`), gomock (`go.uber.org/mock`), testify, swag |
| Frontend | Next.js 15.5.2 (App Router, Turbopack, `output: standalone`), React 19.1, TypeScript 5 strict, alias `@/*` -> `src/*` |
| Frontend testing | Vitest 3 + Testing Library + jsdom, Playwright 1.58.2 |
| Runtime | Node 20 in Docker and CI; npm with `package-lock.json` |
| Infra | Redis 7-alpine (Compose), Helm chart in `helm/`, Docker Hub images |

## High-Level Workflows

Fastest backend loop (Redis required):

```bash
make init              # first time: Go deps + frontend npm deps
make infra-up          # Redis via Compose profile "local"
make run               # go run ./cmd/api on :8080
go test ./internal/... -run TestCreateRoom
```

Fastest frontend loop:

```bash
make run-frontend      # or: cd frontend/planning-poker-front && npm run dev
cd frontend/planning-poker-front && npx vitest run src/hooks/room/useRoomConnection.test.ts
```

Verify a backend change before handback:

```bash
make test-unit         # runs lint + fmt, then go test ./internal/...
make test-integration  # starts/tears down Redis, then test/integration/...
```

Full repository verification:

```bash
make tests                                          # lint, fmt, Redis, go test ./...
cd frontend/planning-poker-front && npm run test:coverage && npm run build
make e2e-local                                      # Docker Playwright flow
```

Full stack locally in Docker:

```bash
make app-compose-up    # redis + backend + frontend
make app-compose-down
```

Release flow (evidence: `.github/workflows/`): merge to `main` runs tests, tags `planning-poker-backend/<semver>` or `planning-poker-frontend/<semver>`, pushes Docker Hub images, and dispatches to the external `bruno303/study-topics` deploy repo. Tag `planning-poker-chart/<version>` publishes the Helm chart to GHCR. Staging deploys run via `workflow_dispatch` on `planning-poker-staging-backend.yml` / `planning-poker-staging-frontend.yml`.

## Development Commands

| Category | Command | Working dir | What it does | Notes |
|---|---|---|---|---|
| Install | `make init` | root | `go mod tidy`, `go mod download`, frontend `npm i` | Auto-copies `example.env` to `.env` when missing |
| Run API | `make run` | root | `go run ./cmd/api` | Needs Redis on `REDIS_HOST:REDIS_PORT` |
| Run frontend | `make run-frontend` | root | Next.js dev server on :3000 | `npm run dev:debug` for inspector |
| Build backend | `make build` | root | `bin/api` | |
| Build frontend | `npm run build` | `frontend/planning-poker-front` | Next production build (standalone) | Dockerfile runs this |
| Lint | `make lint` | root | `golangci-lint run` | Backend only; no frontend lint script |
| Format | `make fmt` | root | `golangci-lint fmt` | Prerequisite of `test-unit`, `tests`, `test-integration` |
| Generate | `make generate` | root | `go generate ./...` | Regenerates `mocks.go` files and Swagger docs |
| Unit tests | `make test-unit` | root | lint, fmt, `go test ./internal/...` | |
| All Go tests | `make tests` | root | lint, fmt, Redis, `go test ./...` | Real target is `tests`, not `test` |
| Integration tests | `make test-integration` | root | lint, fmt, Redis, `test/integration/...` | |
| Coverage | `make test-coverage` | root | `coverage.out` + `coverage.html` | Sonar reads `coverage.out` |
| Single Go test | `go test ./internal/... -run <Name>` | root | One unit test | Integration: `go test ./test/integration/... -run <Name>` |
| Frontend tests | `npm run test` | `frontend/planning-poker-front` | `vitest run` | Coverage: `npm run test:coverage` (lcov) |
| Single FE test | `npx vitest run <path>` | `frontend/planning-poker-front` | One Vitest file | |
| E2E | `make e2e-local` | root | Builds stack, runs Playwright in Docker, always tears down | Preferred over host `npm run e2e` |
| E2E (CI) | `make e2e-ci` | root | Same flow with `CI=1`, line reporter | |
| Infra | `make infra-up` / `make infra-down` | root | Compose profile `local` (Redis) | |
| Full app | `make app-compose-up` / `make app-compose-down` | root | Compose profile `app` | |
| Helm lint | `helm lint .` | `helm` | Chart validation | |

## Architecture

| Directory | Role and key files | Relationships |
|---|---|---|
| `cmd/api` | Entrypoint: `main.go` loads config, logging/metrics/tracing, router, CORS, serves `API_BACKEND_PORT` | Calls `internal/setup`, `internal/config` |
| `internal/domain` | Contracts `Hub`, `AdminHub`, `Bus` (`hub.go`, `bus.go`), domain errors; `entity/` has `Room`, `Client`, `Story`, `Time` + tests | Depended on by application/infra; imports no infra |
| `internal/application/lock` | `LockManager` interface + generated mocks | Implements room-mutation serialization contract |
| `internal/application/planningpoker/usecase` | One file per use case (`createroom.go`, `vote.go`, `reveal.go`, backlog/story/admin actions), `facade.go`, `dto/commands.go` | Orchestrates domain via `Hub` + `LockManager` |
| `internal/application/planningpoker/metric` | `PlanningPokerMetric` interface (`metric.go`) | Records app metrics from use cases |
| `internal/infra/boundaries/http` | Mux adapters implementing `API` (`Endpoint`/`Methods`/`Handle`), admin auth middleware, Swagger + embedded spec | Maps HTTP/WS requests to use cases |
| `internal/infra/boundaries/hub` | `Hub` adapters: `redis/` (production persistence + pub/sub), `inmemory/` (tests), `clientcollection/` helpers | Implements `domain.Hub` |
| `internal/infra/bus` | `websocketbus.go`: per-client WebSocket read/write loop | Implements `domain.Bus`, created by `WebSocketBusFactory` |
| `internal/infra/lock` | Redis and in-memory `LockManager` implementations | Wired by setup |
| `internal/infra/decorators/usecasedecorators` | `TraceableUseCase` wrappers creating spans | Applied when wiring the facade |
| `internal/setup` | `container.go` (all dependency wiring), `api.go` (route registration), `log.go`, `metric.go`, `trace.go`, `redis.go` | Only package that imports all layers |
| `internal/config` | `Config` struct with `env` + `yaml` tags; loads embedded YAML from `config/` (`embed.go`) | Used by setup and tests (`LoadTestConfig`) |
| `frontend/planning-poker-front/src` | App Router pages (`app/join`, `app/room/[roomId]`), components, `context/` (room, logger, toast), `hooks/room/useRoomConnection.ts`, `components/messages/websocket.ts`, `app/api/logs/route.ts` | Talks to backend over REST/WS only |
| `test/integration` | Full-stack tests via `httptest.Server` (`server.go`, `helpers.go`); `NewTestServer` (Redis) vs `NewInMemoryTestServer` | Uses `setup.NewContainerWithDependencies` |
| `config` | Embedded default configs: `config.yaml`, `config-test.yaml` (`CONFIG_FILE` selects) | Read by `internal/config` |
| `helm` | Chart for backend, frontend, bundled Redis, probes, metrics | Deployed externally via GitOps repo |

Dependency direction: `domain` <- `application` <- `infra`; `setup` wires everything. Never import `internal/infra` from domain or use-case code. Request flow: HTTP/WS adapter -> use case -> `Hub` + `LockManager` -> Redis state/pub-sub -> `BroadcastToRoom` -> WebSocket buses. Use cases mutate rooms under locks; Redis hub propagates changes across API instances.

Reading order for a feature: `cmd/api/main.go` -> `internal/setup/container.go` -> `internal/domain/hub.go` -> one use case (for example `vote.go`) -> `internal/infra/boundaries/hub/redis/hub.go` -> frontend `src/hooks/room/useRoomConnection.ts` + `src/components/messages/websocket.ts`.

## Code Conventions

- Go naming: PascalCase exports; camelCase locals/receivers; DTOs named `<Action>Command` / `<Action>Output` (`usecase/dto/commands.go`).
- Imports grouped stdlib, third-party, local/internal. Let `gofmt`/`golangci-lint fmt` decide formatting.
- Always pass `context.Context` through use cases and infra. Wrap errors with `fmt.Errorf("context: %w", err)`; return errors rather than panicking (panic only for bootstrap failures such as Redis init).
- Use constructor injection and follow existing use-case shape (`Execute(ctx, cmd)`); add new use cases to `UseCasesFacade` and wire them in `internal/setup/container.go`.
- HTTP adapters implement the `API` interface and get registered in `newAPIContainer`; keep DTO mapping at boundaries.
- Tests live beside implementation as `_test.go`; table-driven with `t.Run` where scenarios share setup; `gomock` mocks plus testify.
- Mock files are generated (`//go:generate go tool mockgen ...` in `internal/domain`, `internal/application/lock`, `usecase`, `metric`, `infra/boundaries/http`, `infra/boundaries/hub/redis`). Run `make generate` after interface changes; never hand-edit `mocks.go` or Swagger output.
- Config fields need both `env:` and `yaml:` tags and a matching key in `config/config.yaml`.
- Frontend is strict TypeScript: avoid `any`; named types for props/state/payloads; prefer `@/` imports; PascalCase components, camelCase hooks/functions.
- Keep WebSocket/API payload types in sync with backend DTOs (`src/components/messages/websocket.ts` is the frontend contract).
- Context providers fail loudly for invalid usage (see `useRoom`); do not silence errors affecting room state, votes, or connection lifecycle.
- Keep documentation and comments ASCII-only; preserve the surrounding file style.

## Observability

- Backend logs: go-toolkit slog adapter (`internal/setup/log.go`); `LOG_LEVEL` controls level; JSON output when `ENVIRONMENT` is `production`, `staging`, or `development`.
- Metrics: OTel Prometheus exporter (`internal/setup/metric.go`) on `METRICS_PORT` (:9090, `METRICS_PATH=/metrics`); scrape `http://localhost:9090/metrics`. Use-case timings/counters live in `internal/application/planningpoker/metric`.
- Traces: OTLP gRPC via `TRACE_ENABLED` + `TRACE_OTLP_ENDPOINT` (:4317) and `API_TRACING_ENABLED` (`otelmux` middleware in `cmd/api/main.go`); Helm points at `tempo.monitoring.svc.cluster.local:4317`. `usecasedecorators.NewTraceableUseCase` creates spans per use case.
- Frontend logs: `LogBus` (`src/lib/logger.ts`) batches to `POST /api/logs`, which pushes to Loki when `LOKI_URL` is set; the route returns 503 and the client disables logging when Loki is unconfigured.
- No dashboards, alert rules, or Loki service exist in this repository; they live in the external deploy repo (Needs confirmation).

## Infrastructure and External Resources

- Compose services (`docker-compose.yml`): `redis` (6379, healthcheck), `backend` (:8080, `/health` check, profile `app`), `frontend` (:3000, profile `app`, build args `NEXT_PUBLIC_BACKEND_URL`/`NEXT_PUBLIC_WEBSOCKET_URL`), `frontend-e2e` (profile `e2e`), `playwright` (profile `e2e`).
- Ports/env defaults: `example.env` (`COMPOSE_PROJECT_NAME=planning-poker`, Redis 6379, backend 8080, frontend 3000, `TRACE_OTLP_ENDPOINT=localhost:4317`, `ADMIN_API_KEY` default in `config/config.yaml`).
- Worktree isolation: pass unique `COMPOSE_PROJECT_NAME`, `REDIS_HOST_PORT`, `BACKEND_HOST_PORT`, `FRONTEND_HOST_PORT` to `make app-compose-up` / `app-compose-down` / test targets.
- Redis is mandatory at startup: `newInfraContainer` panics if the Redis hub cannot be created. Production uses Redis for room state, version checks, pub/sub, and distributed locks.
- Frontend env: `frontend/planning-poker-front/example.env` (`NEXT_PUBLIC_BACKEND_URL`, `NEXT_PUBLIC_WEBSOCKET_URL`, `LOKI_URL`, `LOG_ENV`, `NEXT_PUBLIC_LOG_LEVEL`). `NEXT_PUBLIC_*` values are baked in at build time.
- Docker images: `bruno303/planning-poker-backend`, `bruno303/planning-poker-frontend`; Helm chart `oci://ghcr.io/bruno303/planning-poker-chart`.
- CI secrets referenced (do not commit): `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN`, `SONAR_TOKEN`, `PLANNING_POKER_PAT`, `DEEPSEEK_API_KEY` (opencode workflows).
- `.env` is gitignored and auto-created by the Makefile from `example.env`.

## Testing Strategy

- Unit: `internal/**/*_test.go`, run with `make test-unit` or `go test ./internal/... -run <Name>`.
- Integration: `test/integration/**`, real router/middleware/wiring via `httptest.Server`; use `NewInMemoryTestServer` for fast API/WS tests and `NewTestServer` when Redis behavior matters. Helpers and examples: `test/integration/README.md`.
- Frontend unit: Vitest + Testing Library + jsdom, specs colocated as `src/**/*.test.ts(x)`; coverage via v8/lcov.
- E2E: Playwright specs in `frontend/planning-poker-front/e2e/`, run through Docker (`make e2e-local` / `make e2e-ci`); default base URL `http://frontend-e2e:3000`. Details: `e2e/README.md`.
- CI (`planning-poker-ci.yml`, `planning-poker-frontend-ci.yml`, `planning-poker-e2e-ci.yml`): backend `go test -count=1 -timeout 30s -coverpkg=./internal/... -coverprofile=coverage.out ./...` with a Redis service plus golangci-lint; frontend `npm ci --ignore-scripts` + `npm run test:coverage`; E2E `make e2e-ci`; both backend and frontend upload coverage to SonarQube and enforce the quality gate (no local threshold configured).
- `make test-unit`, `make tests`, and `make test-integration` run `lint` and `fmt` first; expect formatting rewrites and a lint failure gate.

## Agent Working Rules

- Search first (`grep`/`rg`) and read only relevant files; ignore `vendor/`, `bin/`, `node_modules/`, `.next/`, `coverage.*`.
- Read `test/integration/README.md` and `frontend/planning-poker-front/e2e/README.md` before touching those suites.
- Prefer minimal, targeted changes that mirror existing patterns; do not introduce new layers when an existing package owns the concern.
- Verify with the fastest relevant check first (single test file), then `make test-unit` or `npm run test`; use `make tests` plus `make e2e-local` for broad changes.
- Run `make generate` after changing interfaces, mocks, or HTTP API docs; regenerate rather than hand-edit generated files.
- Keep frontend WebSocket message types aligned with backend payloads; frontend type errors surface through `npm run build`.
- Update `README.MD` or the relevant directory README when structure or commands change.

## Known Gaps or Unknowns

- `.github/copilot-instructions.md` does not exist in this repo (not on `origin/main`); there are no Cursor or Copilot rule files. Follow this AGENTS.md and executable configs.
- No frontend lint/typecheck npm script; CI runs only Vitest coverage, so type errors are caught by `npm run build` (Needs confirmation whether CI type-checks explicitly).
- No Loki service, Grafana dashboards, or alert rules in this repo; `LOKI_URL` and dashboards come from external infrastructure (Needs confirmation).
- README states Go 1.24+ but `go.mod`, Dockerfile, and CI pin 1.25.4; follow the pinned version.
- Node version is not declared in `package.json` (`engines` absent); Docker/CI use Node 20, so match that locally.
- `.planning/` and `.agents/` are gitignored; content conventions unknown.
- `make deps` vendors Go modules into gitignored `vendor/`; not required for normal development.
