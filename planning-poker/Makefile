FRONT_DIR=frontend/planning-poker-front
BACKEND_URL=http://localhost:8080
WEBSOCKET_URL=ws://localhost:8080

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

.PHONY: test-unit
test-unit: lint fmt
	go test -v -timeout 30s ./internal/...

.PHONY: test-integration
test-integration: lint fmt
	go test -v -timeout 30s ./test/integration/...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out -o coverage.html
	echo "Generated coverage report at coverage.html"
	echo "If using WSL, run 'explorer.exe coverage.html' to view the report."

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: generate
generate:
	go generate ./...

.PHONY: clean
clean:
	rm -rf bin
	rm -rf coverage.out
	rm -rf coverage.html
