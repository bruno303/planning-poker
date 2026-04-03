Common commands (repo root unless noted):
- make init (install Go deps + frontend npm deps)
- make run (run backend)
- make run-frontend (run frontend dev)
- make build (build backend)
- make tests (lint+fmt+go test ./...)
- make test-unit
- make test-integration
- make lint
- make fmt
- make generate
- make infra-up / make infra-down
Frontend dir (frontend/planning-poker-front):
- npm run dev
- npm run dev:debug
- npm run build
- npm run start
- npm run test (vitest run)
- npm run e2e (playwright test)
Note: real all-tests target is `make tests` (not `make test`).