Place end-to-end Playwright specs in this directory.

The default base URL is `http://frontend-e2e:3000` (Docker Compose E2E frontend service endpoint).
Override it with `PLAYWRIGHT_BASE_URL` when needed.

## Compose-based E2E prerequisites

- Docker and Docker Compose installed and running.
- Ports `3000` (frontend), `8080` (backend), and `6379` (redis) available on the host.
- No host Playwright browser/system libraries are required; tests run inside a Playwright Docker image.

## Local run

From the repository root:

```bash
make e2e-local
```

## CI invocation

From the repository root:

```bash
make e2e-ci
```

## Startup and cleanup behavior

- E2E brings app services up with Docker Compose profiles `app` and `e2e`, explicitly starting `redis`, `backend`, `frontend`, and `frontend-e2e`.
- `frontend` is built for host-browser usage (`localhost` backend/websocket URLs), while `frontend-e2e` is built for Docker-internal addressing (`backend` service URL).
- Playwright itself runs in a separate `playwright` container using profile `e2e` when `make e2e-local` / `make e2e-ci` invokes the runner.
- The Playwright runner container uses host-mapped UID/GID (`HOST_UID`/`HOST_GID`, default `1000:1000`) to avoid root-owned artifacts on the mounted workspace.
- Playwright dependencies are installed inside a Docker-managed volume mounted at `/work/node_modules` (not in host-managed `node_modules`).
- Before executing Playwright, the E2E make targets ensure writable ownership for `/work/node_modules`, `/work/test-results`, and `/work/playwright-report` matching the host-mapped UID/GID.
- The flow waits for service readiness via `docker compose ... up -d --wait` before launching Playwright.
- Readiness depends on service health checks; first startup can take longer while images build and services warm up.
- Cleanup is expected on every run via `make e2e-compose-down` (`down --remove-orphans --volumes`), including after failures.
- If a run is interrupted, execute `make e2e-compose-down` manually before retrying.

## Test discovery behavior

- The default `npm run e2e` command fails when no specs are discovered.
