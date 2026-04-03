FRONT_DIR=frontend/planning-poker-front
BACKEND_URL=http://localhost:8080
WEBSOCKET_URL=ws://localhost:8080
DOCKER_COMPOSE=docker compose
COMPOSE_APP_PROFILE=app
COMPOSE_E2E_PROFILE=e2e
E2E_READY_MAX_RETRIES ?= 60
HOST_UID=$(shell id -u)
HOST_GID=$(shell id -g)

ifeq ($(wildcard .env),)
$(shell cp example.env .env)
endif

# Only load .env for non-test targets
ifeq ($(filter test-integration,$(MAKECMDGOALS)),)
include .env
export
endif

.PHONY: init
init:
	go mod tidy
	go mod download
	cd $(FRONT_DIR) && npm i

.PHONY: deps
deps:
	go mod tidy
	go mod vendor

.PHONY: download
download:
	go mod tidy
	go mod download

.PHONY: run
run:
	go run ./cmd/api

.PHONY: run-frontend
run-frontend:
	cd $(FRONT_DIR) && npm run dev

.PHONY: build
build:
	go build -o bin/api ./cmd/api

.PHONY: tests
tests: lint fmt
	go test -timeout 30s ./...

.PHONY: test-unit
test-unit: lint fmt
	go test -timeout 30s ./internal/...

.PHONY: test-integration
test-integration: lint fmt
	go test -timeout 30s ./test/integration/...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out -o coverage.html
	echo "Generated coverage report at coverage.html"
	echo "If using WSL, run 'explorer.exe coverage.html' to view the report."

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: lint
lint:
	golangci-lint run

.PHONY: generate
generate:
	go generate ./...

.PHONY: clean
clean:
	rm -rf bin
	rm -rf coverage.out
	rm -rf coverage.html

.PHONY: infra-up
infra-up:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) up -d

.PHONY: infra-down
infra-down:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) down

.PHONY: e2e-compose-up
e2e-compose-up:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) up -d --build redis backend frontend-e2e

.PHONY: e2e-compose-wait
e2e-compose-wait:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) up -d --wait redis backend frontend-e2e

.PHONY: e2e-compose-down
e2e-compose-down:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) down --remove-orphans --volumes

.PHONY: e2e-playwright
e2e-playwright:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps --user root playwright sh -lc "mkdir -p /work/node_modules /work/test-results /work/playwright-report && chown -R $(HOST_UID):$(HOST_GID) /work/node_modules /work/test-results /work/playwright-report" && HOST_UID=$(HOST_UID) HOST_GID=$(HOST_GID) $(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps playwright sh -lc "npm ci && attempt=0; until node -e \"fetch('http://frontend-e2e:3000').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))\"; do attempt=$$((attempt + 1)); if [ $$attempt -ge $(E2E_READY_MAX_RETRIES) ]; then echo 'frontend-e2e did not become ready in time'; exit 1; fi; sleep 1; done && npm run e2e"

.PHONY: e2e-playwright-ci
e2e-playwright-ci:
	$(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps --user root playwright sh -lc "mkdir -p /work/node_modules /work/test-results /work/playwright-report && chown -R $(HOST_UID):$(HOST_GID) /work/node_modules /work/test-results /work/playwright-report" && HOST_UID=$(HOST_UID) HOST_GID=$(HOST_GID) $(DOCKER_COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps playwright sh -lc "npm ci && attempt=0; until node -e \"fetch('http://frontend-e2e:3000').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))\"; do attempt=$$((attempt + 1)); if [ $$attempt -ge $(E2E_READY_MAX_RETRIES) ]; then echo 'frontend-e2e did not become ready in time'; exit 1; fi; sleep 1; done && CI=1 npm run e2e -- --reporter=line"

.PHONY: e2e-local
e2e-local:
	@status=0; \
	$(MAKE) e2e-compose-up || status=$$?; \
	if [ $$status -eq 0 ]; then $(MAKE) e2e-compose-wait || status=$$?; fi; \
	if [ $$status -eq 0 ]; then $(MAKE) e2e-playwright || status=$$?; fi; \
	$(MAKE) e2e-compose-down || true; \
	exit $$status

.PHONY: e2e-ci
e2e-ci:
	@status=0; \
	$(MAKE) e2e-compose-up || status=$$?; \
	if [ $$status -eq 0 ]; then $(MAKE) e2e-compose-wait || status=$$?; fi; \
	if [ $$status -eq 0 ]; then $(MAKE) e2e-playwright-ci || status=$$?; fi; \
	$(MAKE) e2e-compose-down || true; \
	exit $$status
