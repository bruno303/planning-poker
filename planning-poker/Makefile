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
test:
	go test -timeout 30s ./...

.PHONY: clean
clean:
	rm -rf bin

.PHONY: fmt
fmt:
	go fmt ./...

generate:
	go generate ./...

docker-build-backend:
	docker build --no-cache -t planning-poker-backend .

docker-build-frontend:
	cd $(FRONT_DIR) && docker build -f Dockerfile \
		--build-arg NEXT_PUBLIC_BACKEND_URL=$(BACKEND_URL) \
  	--build-arg NEXT_PUBLIC_WEBSOCKET_URL=$(WEBSOCKET_URL) \
		--no-cache -t planning-poker-frontend .
