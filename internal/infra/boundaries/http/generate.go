package http

//go:generate go tool mockgen -destination mocks.go -typed -package http . API
//go:generate go tool swag init -g ../../../../cmd/api/main.go -o swagger --outputTypes json,yaml --parseInternal
