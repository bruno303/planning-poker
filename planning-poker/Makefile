.PHONY: init
init:
	go mod tidy
	go mod download
	npm install -g http-server

.PHONY: run
run:
	go run ./cmd/api

.PHONY: run-frontend
run-frontend:
	cd frontend && http-server -p 3000

.PHONY: build
build:
	go build -o bin/api ./cmd/api

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -rf bin

.PHONY: fmt
fmt:
	go fmt ./...
