# Start from the official Golang image for building
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod downloads
RUN apk add --no-cache git

# Copy go.mod and go.sum first for caching dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
RUN go build -o app ./cmd/api

# Use a minimal image for running
FROM alpine:latest

WORKDIR /app

# Copy the built binary from builder
COPY --from=builder /app/app .

# Expose port (change if your app uses a different port)
EXPOSE 8080

# Run the binary
CMD ["./app"]
