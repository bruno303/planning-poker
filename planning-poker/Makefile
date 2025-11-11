FRONT_DIR=frontend/planning-poker-front
BACKEND_URL=http://localhost:8080
WEBSOCKET_URL=ws://localhost:8080

ifeq ($(wildcard .env),)
$(shell cp example.env .env)
endif

include .env
export

.PHONY: init
init:
	go mod tidy
	go mod download
	cd $(FRONT_DIR) && npm i

.PHONY: deps
deps:
	go mod tidy
	go mod vendor

.PHONY: run
run:
	go run ./cmd/api

.PHONY: run-frontend
run-frontend:
	cd $(FRONT_DIR) && npm run dev

.PHONY: build
build:
	go build -o bin/api ./cmd/api

.PHONY: test
test: lint fmt
	go test -timeout 30s ./...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	echo "Generated coverage report at coverage.html"
	echo "If using WSL, run 'explorer.exe coverage.html' to view the report."

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: clean
clean:
	rm -rf bin

.PHONY: generate
generate:
	go generate ./...

.PHONY: docker-build-backend
docker-build-backend: test
	docker build -t planning-poker-backend -f Dockerfile .

.PHONY: docker-build-frontend
docker-build-frontend:
	cd $(FRONT_DIR) && docker build -f Dockerfile \
		--build-arg NEXT_PUBLIC_BACKEND_URL=$(BACKEND_URL) \
  	--build-arg NEXT_PUBLIC_WEBSOCKET_URL=$(WEBSOCKET_URL) \
		-t planning-poker-frontend .
