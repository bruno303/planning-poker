# Start from the official Golang image for building
FROM golang:1.25.4-alpine AS builder

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

# Use a pinned minimal image for running
FROM alpine:3.21.3

WORKDIR /app

# Run the application without root privileges.
RUN addgroup -S app && adduser -S app -G app

# Copy the built binary from builder
COPY --from=builder --chown=app:app /app/app .

USER app

# Expose port 8080 for the application
EXPOSE 8080

# Run the binary
CMD ["./app"]
