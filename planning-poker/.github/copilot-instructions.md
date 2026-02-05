# Planning Poker - AI Agent Instructions

## Architecture Overview

This is a real-time Planning Poker application with **Go backend + Next.js frontend**, following **Clean Architecture** principles with DDD patterns.

### Layer Structure (Go Backend)

- `cmd/api/` - Application entry point; dependency injection via [container.go](../cmd/api/container.go)
- `internal/domain/` - Core business logic (entities, interfaces). Example: [Room](../internal/domain/entity/room.go) entity with voting rules
- `internal/application/` - Use cases implementing business workflows (see [usecase/](../internal/application/planningpoker/usecase/))
- `internal/infra/` - External concerns (HTTP handlers, WebSockets, Redis storage)

**Key Pattern**: Dependencies point inward. Domain defines interfaces (`Hub`, `Bus`), infra implements them (`RedisHub`).

## Critical Workflows

### Mock Generation

Run `make generate` to regenerate mocks after changing interfaces. Uses `go:generate` directives:

```go
//go:generate go tool mockgen -destination mocks.go -typed -package domain . Hub,AdminHub
```

Examples in [internal/domain/generate.go](../internal/domain/generate.go), [internal/application/lock/generate.go](../internal/application/lock/generate.go).

### Testing

- `make test` - Runs lint, fmt, and all tests (unit + integration) with 30s timeout
- `make test-unit` - Runs only unit tests in `internal/` directory
- `make test-integration` - Runs only integration tests in `test/integration/`
- `make test-coverage` - Generates HTML coverage report at `coverage.html`
- Tests use `go.uber.org/mock` (see [createroom_test.go](../internal/application/planningpoker/usecase/createroom_test.go))

**Integration Tests**: Use `httptest.Server` with full app stack. See [test/integration/](../test/integration/) for setup and examples.

### Development

```bash
make init          # Install all deps (Go + npm)
make run           # Start backend on :8080
make run-frontend  # Start Next.js on :3000 (separate terminal)
docker-compose up  # Run both services containerized
```

## Use Case Pattern

All business logic uses generic use case interfaces:

- `UseCase[In]` - No return value (e.g., vote, toggle spectator)
- `UseCaseR[In, Out]` - Returns result (e.g., create room, join room)

See [internal/application/usecase.go](../internal/application/usecase.go). Each use case wraps with tracing decorator in [container.go](../cmd/api/container.go#L62-L85).

**Example**: [CreateRoomUseCase](../internal/application/planningpoker/usecase/createroom.go) → wrapped with `TraceableUseCaseR` → exposed via facade.

## WebSocket Communication

Real-time updates use WebSocket at `/planning/ws/{roomId}`. Messages follow command pattern:

```json
{"type": "vote", "roomId": "...", "clientId": "...", "vote": "5"}
{"type": "reveal-votes", "roomId": "...", "clientId": "..."}
```

See [websocket.go](../internal/infra/boundaries/http/websocket.go) for handler mapping and [dto/commands.go](../internal/application/planningpoker/usecase/dto/commands.go) for message formats.

**Broadcasting**: `Hub.BroadcastToRoom()` sends state updates to all connected clients via their `Bus` instances.

## Domain Model

### Room Entity

- Manages participants, voting state, reveal logic
- Auto-promotes new owner when last owner leaves (see [RemoveClient](../internal/domain/entity/room.go#L57-L67))
- Calculates average/mode on reveal

### Client Entity

- Bidirectional reference to Room (`client.room`)
- States: owner, spectator, voted

### Hub (Redis-based)

Central registry managing rooms, clients, and WebSocket buses. **Stateless** - all room state persists in Redis. Uses Redis pub/sub to broadcast updates across multiple backend instances. See [redis/hub.go](../internal/infra/boundaries/bus/redis/hub.go).

**Key Features**:

- Room state serialized to Redis with 24-hour TTL
- Pub/sub channels (`planning-poker:updates:{roomId}`) broadcast state changes
- Each instance maintains local WebSocket `Bus` map for connected clients
- Supports horizontal scaling - multiple backend instances can serve the same room

**In-Memory Alternative**: [inmemory/hub.go](../internal/infra/boundaries/bus/inmemory/hub.go) available for testing/single-instance deployments.

## Configuration

Uses `sethvargo/go-envconfig` + YAML. Load via [config.LoadConfig()](../internal/config/config.go).

Environment vars override YAML (see [example.env](../example.env)):

```
API_BACKEND_PORT=8080
API_CORS_ALLOWED_ORIGINS=http://localhost:3000
TRACE_ENABLED=false
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

## Redis Configuration

The application uses Redis for stateless operation:

- **Room persistence**: Serialized room state with 24-hour TTL
- **Pub/sub broadcasting**: Updates distributed across instances via channels
- **Client-room mapping**: Tracks which clients belong to which rooms

Required configuration (via environment or YAML):

- `REDIS_HOST`: Redis server host (default: localhost)
- `REDIS_PORT`: Redis server port (default: 6379)
- `REDIS_PASSWORD`: Redis password (optional)
- `REDIS_DB`: Redis database number (default: 0)

Start Redis locally: `docker run -p 6379:6379 redis:7-alpine` or use docker-compose.

## Frontend Conventions

Next.js 15 with App Router (Turbopack):

- Pages: `/app/join/[roomId]/` (join flow), `/app/room/[roomId]/` (main room UI)
- Context: [RoomContext](../frontend/planning-poker-front/src/context/room/roomContext.tsx) manages WebSocket lifecycle
- Commands: TypeScript matches backend DTOs (vote, reveal, toggle-spectator, etc.)

Start dev server with `make run-frontend` or `npm run dev` in `frontend/planning-poker-front/`.

## Lock Manager Pattern

[LockManager](../internal/application/lock/lock.go) prevents race conditions in concurrent use cases (vote, reveal). In-memory implementation uses sync primitives.

## Observability

- OpenTelemetry tracing (optional via config)
- Use cases decorated with [TraceableUseCase](../internal/infra/decorators/usecasedecorators/trace.go)
- Prometheus metrics via `bruno303/go-toolkit`

## When Adding Features

1. Define interface in `internal/domain/` if cross-layer
2. Implement use case in `internal/application/planningpoker/usecase/`
3. Add HTTP handler in `internal/infra/boundaries/http/`
4. Wire in [internal/setup/container.go](../internal/setup/container.go)
5. Run `make generate` if new interfaces added
6. Add tests (pattern: `<feature>_test.go` alongside implementation)
7. Consider adding integration test in `test/integration/` for end-to-end validation
