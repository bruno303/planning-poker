# AGENTS.md

Repository guide for coding agents working in `planning-poker`.

## Scope

- Backend: Go 1.25.x API in `cmd/api`, `internal`, and `test/integration`.
- Frontend: Next.js 15 app in `frontend/planning-poker-front` with strict TypeScript.
- Architecture is clean-ish and layered: domain -> application -> infra.
- Keep changes repository-specific, small, and consistent with existing patterns.

## Agent Rule Sources

- Copilot rules exist in `.github/copilot-instructions.md`.
- No Cursor rules were found in this repository.
- If guidance conflicts, prefer the actual code and executable commands in the repo.

## Quick Paths

- API entrypoint: `cmd/api/main.go`
- Dependency wiring: `internal/setup/container.go`
- Domain contracts and entities: `internal/domain`, `internal/domain/entity`
- Use cases: `internal/application/planningpoker/usecase`
- Infra adapters: `internal/infra`
- Frontend app: `frontend/planning-poker-front/src`
- Integration tests: `test/integration`

## Verified Commands

Run from the repository root unless noted.

### Setup and dependencies

- `make init` - install Go deps and frontend npm deps.
- `make deps` - tidy and vendor Go modules.
- `make download` - tidy and download Go modules.

### Run

- `make run` - run the backend API.
- `make run-frontend` - run the Next.js frontend.
- `npm run dev` - same frontend dev flow from `frontend/planning-poker-front`.
- `npm run dev:debug` - frontend dev mode with Node inspector.

### Build

- `make build` - build backend binary to `bin/api`.
- `npm run build` - build the frontend from `frontend/planning-poker-front`.
- `npm run start` - start the built frontend from `frontend/planning-poker-front`.
- `npm run start:debug` - start the built frontend with inspector.

### Lint, format, generate

- `make lint` - run `golangci-lint`.
- `make fmt` - run `golangci-lint fmt`.
- `make generate` - run `go generate ./...`.

Regenerate after interface changes, especially when mocks are generated from `go:generate` directives.

### Tests

- `make tests` - real all-tests target; runs lint, fmt, then `go test ./...`.
- `make test-unit` - unit tests under `./internal/...`.
- `make test-integration` - integration tests under `./test/integration/...`.
- `make test-coverage` - generate `coverage.out` and `coverage.html`.

Important: the real Makefile target is `make tests`, not `make test`.
`.github/copilot-instructions.md` mentions `make test`, but that target does not exist in the Makefile.

### Single test commands

- Single Go unit test: `go test ./internal/... -run TestCreateRoomUseCase_Execute_Success`
- Single Go integration test: `go test ./test/integration/... -run TestHealthcheck`
- Verbose integration run: `go test -v ./test/integration/...`

### Infra

- `make infra-up` - start local infra with Docker Compose.
- `make infra-down` - stop local infra with Docker Compose.

## Backend Go Conventions

### Architecture boundaries

- Keep domain logic in `internal/domain` and `internal/domain/entity`.
- Keep orchestration in `internal/application/planningpoker/usecase`.
- Keep framework, transport, Redis, WebSocket, and other adapters in `internal/infra`.
- Wire dependencies in `internal/setup/container.go`.
- Depend inward: domain defines interfaces, infra implements them.
- Do not import infra packages from domain or use-case code.

### Naming and structure

- Exported names use PascalCase.
- Local vars, receivers, and helpers use camelCase.
- Use explicit command/output DTO naming such as `CreateRoomCommand` and `CreateRoomOutput`.
- Keep file names lowercase and aligned with the feature or use case.
- Put tests beside implementation with `_test.go` suffix.

### Imports and formatting

- Group imports as stdlib, third-party, then local/internal.
- Let `gofmt`/`golangci-lint fmt` control formatting; do not hand-format for style.
- Keep imports minimal and remove unused ones.

### Error handling and context

- Pass `context.Context` through use cases and infra calls.
- Return errors instead of panicking, except for startup/bootstrap failures and explicit test setup shortcuts.
- Wrap errors with context using `fmt.Errorf("context: %w", err)`.
- Prefer clear error returns over hidden logging-only failures.

### Use cases and dependencies

- Follow existing use-case patterns in `internal/application/planningpoker/usecase`.
- Use constructor injection.
- If a dependency crosses layers, define its interface in the domain or application contract layer, not infra.
- Keep DTO mapping at boundaries; do not leak storage or transport models into domain logic.

## Backend Testing Conventions

- Unit tests live next to the code they cover.
- Integration tests live in `test/integration` and use the full app stack.
- Prefer table-driven tests when multiple scenarios share setup.
- Use `t.Run` for named subtests.
- Use `gomock` for generated mocks and `testify` where already established.
- Keep assertions readable and failure messages specific.
- Avoid panics in tests unless validating panic behavior.

### Mocks

- Generated mocks already exist in several packages as `mocks.go`.
- When you change an interface used by mocks, run `make generate`.
- Regenerate mocks before pushing changes that touch `internal/domain`, `internal/application/lock`, or other packages with `go:generate` mock directives.
- Do not hand-edit generated mock files unless the repository clearly treats them as source.

## Frontend Next.js and TypeScript Conventions

- The frontend uses strict TypeScript; keep new code type-safe and avoid `any`.
- Use the `@/*` path alias for imports from `src`.
- Prefer named types for non-trivial props, state, and payloads.
- Keep React components and hooks focused; move shared logic into hooks, context, or typed helpers when needed.
- Match frontend message shapes to backend DTOs for WebSocket and API payloads.

### Frontend naming and imports

- Components use PascalCase.
- Hooks, functions, and variables use camelCase.
- Keep imports tidy and stable; prefer absolute `@/` imports over deep relative chains when importing from `src`.
- Preserve existing semicolon and quote style used in the frontend.

### Frontend error handling

- Fail loudly for invalid provider usage or impossible states, as existing context code does.
- Handle network and socket failures explicitly in UI-facing flows.
- Do not silence errors that affect room state, votes, or connection lifecycle.

## Working Style for Agents

- Read existing files before editing; mirror established patterns.
- Prefer minimal, targeted changes over broad refactors.
- Do not invent new layers when the existing structure already has a home for the change.
- If you add or change interfaces, check whether generated mocks must be refreshed.
- If a doc or instruction disagrees with the repo, document the mismatch and follow the executable source of truth.

## Practical Notes

- Backend startup depends on Redis via the configured hub and lock manager.
- Integration tests use `httptest.Server` and production-like container wiring.
- Frontend scripts live only in `frontend/planning-poker-front/package.json`; there is no frontend lint or test script defined there today.
- Keep documentation and agent guidance ASCII-only unless a file already requires Unicode.
