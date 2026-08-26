FRONT_DIR=frontend/planning-poker-front
BACKEND_URL=http://localhost:$(BACKEND_HOST_PORT)
WEBSOCKET_URL=ws://localhost:$(BACKEND_HOST_PORT)
DOCKER_COMPOSE=docker compose
COMPOSE_APP_PROFILE=app
COMPOSE_LOCAL_PROFILE=local
COMPOSE_E2E_PROFILE=e2e
E2E_READY_MAX_RETRIES ?= 60
HOST_UID=$(shell id -u)
HOST_GID=$(shell id -g)

COMPOSE_PROJECT_NAME_ORIGIN := $(origin COMPOSE_PROJECT_NAME)
COMPOSE_PROJECT_NAME_VALUE := $(COMPOSE_PROJECT_NAME)
REDIS_HOST_PORT_ORIGIN := $(origin REDIS_HOST_PORT)
REDIS_HOST_PORT_VALUE := $(REDIS_HOST_PORT)
BACKEND_HOST_PORT_ORIGIN := $(origin BACKEND_HOST_PORT)
BACKEND_HOST_PORT_VALUE := $(BACKEND_HOST_PORT)
FRONTEND_HOST_PORT_ORIGIN := $(origin FRONTEND_HOST_PORT)
FRONTEND_HOST_PORT_VALUE := $(FRONTEND_HOST_PORT)

ifeq ($(wildcard .env),)
$(shell cp example.env .env)
endif

# Only load .env for non-test targets
ifeq ($(filter test-integration,$(MAKECMDGOALS)),)
include .env
export
endif

ifneq ($(filter environment command line,$(COMPOSE_PROJECT_NAME_ORIGIN)),)
COMPOSE_PROJECT_NAME := $(COMPOSE_PROJECT_NAME_VALUE)
endif
ifneq ($(filter environment command line,$(REDIS_HOST_PORT_ORIGIN)),)
REDIS_HOST_PORT := $(REDIS_HOST_PORT_VALUE)
endif
ifneq ($(filter environment command line,$(BACKEND_HOST_PORT_ORIGIN)),)
BACKEND_HOST_PORT := $(BACKEND_HOST_PORT_VALUE)
endif
ifneq ($(filter environment command line,$(FRONTEND_HOST_PORT_ORIGIN)),)
FRONTEND_HOST_PORT := $(FRONTEND_HOST_PORT_VALUE)
endif
COMPOSE_PROJECT_NAME ?= planning-poker
REDIS_HOST_PORT ?= 6379
BACKEND_HOST_PORT ?= 8080
FRONTEND_HOST_PORT ?= 3000

COMPOSE=$(DOCKER_COMPOSE) --project-name $(COMPOSE_PROJECT_NAME)

export COMPOSE_PROJECT_NAME REDIS_HOST_PORT BACKEND_HOST_PORT FRONTEND_HOST_PORT

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
	@status=0; \
	$(MAKE) test-infra-up || status=$$?; \
	if [ $$status -eq 0 ]; then REDIS_HOST=localhost REDIS_PORT=$(REDIS_HOST_PORT) go test -timeout 30s -count=1 ./... || status=$$?; fi; \
	$(MAKE) test-infra-down || true; \
	exit $$status

.PHONY: test-unit
test-unit: lint fmt
	go test -timeout 30s -count=1 ./internal/...

.PHONY: test-integration
test-integration: lint fmt
	@status=0; \
	$(MAKE) test-infra-up || status=$$?; \
	if [ $$status -eq 0 ]; then REDIS_HOST=localhost REDIS_PORT=$(REDIS_HOST_PORT) go test -timeout 30s -count=1 ./test/integration/... || status=$$?; fi; \
	$(MAKE) test-infra-down || true; \
	exit $$status

.PHONY: test-coverage
test-coverage:
	@status=0; \
	$(MAKE) test-infra-up || status=$$?; \
	if [ $$status -eq 0 ]; then REDIS_HOST=localhost REDIS_PORT=$(REDIS_HOST_PORT) go test -count=1 -coverprofile=coverage.out -coverpkg=./internal/... ./internal/... ./test/integration/... || status=$$?; fi; \
	$(MAKE) test-infra-down || true; \
	if [ $$status -eq 0 ]; then go tool cover -html=coverage.out -o coverage.html; echo "Generated coverage report at coverage.html"; fi; \
	exit $$status

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
	$(COMPOSE) --profile $(COMPOSE_LOCAL_PROFILE) up -d --wait

.PHONY: infra-down
infra-down:
	$(COMPOSE) --profile $(COMPOSE_LOCAL_PROFILE) down

.PHONY: test-infra-up
test-infra-up:
	$(COMPOSE) --profile $(COMPOSE_LOCAL_PROFILE) up -d --wait redis

.PHONY: test-infra-down
test-infra-down:
	$(COMPOSE) --profile $(COMPOSE_LOCAL_PROFILE) down

.PHONY: app-compose-up
app-compose-up:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) up -d --build

.PHONY: app-compose-down
app-compose-down:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) down

.PHONY: e2e-compose-up
e2e-compose-up:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) up -d --build redis backend frontend-e2e

.PHONY: e2e-compose-wait
e2e-compose-wait:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) up -d --wait redis backend frontend-e2e

.PHONY: e2e-compose-down
e2e-compose-down:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) down --remove-orphans --volumes

.PHONY: e2e-playwright
e2e-playwright:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps --user root playwright sh -lc "mkdir -p /work/node_modules /work/test-results /work/playwright-report && chown -R $(HOST_UID):$(HOST_GID) /work/node_modules /work/test-results /work/playwright-report" && $(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps --user root playwright sh -lc "npm ci --cache /tmp/npm-cache && attempt=0; until node -e \"fetch('http://frontend-e2e:3000').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))\"; do attempt=$$((attempt + 1)); if [ $$attempt -ge $(E2E_READY_MAX_RETRIES) ]; then echo 'frontend-e2e did not become ready in time'; exit 1; fi; sleep 1; done && npm run e2e"

.PHONY: e2e-playwright-ci
e2e-playwright-ci:
	$(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps --user root playwright sh -lc "mkdir -p /work/node_modules /work/test-results /work/playwright-report && chown -R $(HOST_UID):$(HOST_GID) /work/node_modules /work/test-results /work/playwright-report" && $(COMPOSE) --profile $(COMPOSE_APP_PROFILE) --profile $(COMPOSE_E2E_PROFILE) run --rm --no-deps --user root playwright sh -lc "npm ci --cache /tmp/npm-cache && attempt=0; until node -e \"fetch('http://frontend-e2e:3000').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))\"; do attempt=$$((attempt + 1)); if [ $$attempt -ge $(E2E_READY_MAX_RETRIES) ]; then echo 'frontend-e2e did not become ready in time'; exit 1; fi; sleep 1; done && CI=1 npm run e2e -- --reporter=line"

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
