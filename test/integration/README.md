# Integration Tests

This directory contains integration tests that verify the application works correctly when all components are wired together.

## Running Tests

```bash
# Run all integration tests with the matching local Redis infrastructure
make test-integration

# Run directly when Redis is already available
go test ./test/integration/...

# Run specific test
go test ./test/integration/... -run TestHealthcheck

# Run with verbose output
go test -v ./test/integration/...

# Run with coverage
go test -cover ./test/integration/...
```

## Test Structure

- `server.go` - Server setup
- `helpers.go` - Common testing utilities and HTTP client helpers
- `README.md` - This file

## Writing Integration Tests

Integration tests use `httptest.Server` to spin up a real HTTP server with all middleware and dependencies configured. This ensures tests reflect actual production behavior.

There are two server modes:

- `NewTestServer(t)` uses the production Redis hub and lock manager. Use it for tests that verify Redis adapters, and run Redis before invoking `go test` directly.
- `NewInMemoryTestServer(t)` uses `InMemoryHub` and `InMemoryLockManager`. Use it for fast full-stack API and WebSocket tests that do not require Redis.

The Make target starts the current Compose project with the local profile,
passes its Redis host port through `REDIS_HOST` and `REDIS_PORT`, and always
tears that project down after the test command. Direct `go test` runs support
`REDIS_HOST`, `REDIS_PORT`, and `REDIS_DB` environment overrides.

### Example Test

```go
func TestMyEndpoint(t *testing.T) {
    ts := NewTestServer(t)
    defer ts.Close()

    t.Run("description", func(t *testing.T) {
        var response MyResponse
        resp, err := ts.GetJSON(t, "/my-endpoint", &response)
        if err != nil {
            t.Fatalf("request failed: %v", err)
        }

        AssertStatus(t, resp, http.StatusOK)
        // Add your assertions here
    })
}
```

### Test Helpers

- `NewTestServer(t)` - Creates a fully configured Redis-backed test server
- `NewInMemoryTestServer(t)` - Creates a fully configured in-memory test server
- `ts.GetJSON(t, path, target)` - GET request with JSON decode
- `AssertStatus(t, resp, expected)` - Assert HTTP status code
- `HTTPClient` - Reusable HTTP client for more complex scenarios

## Best Practices

1. **Isolation**: Each test should be independent and not rely on state from other tests
2. **Cleanup**: Always defer `ts.Close()` to clean up resources
3. **Real Dependencies**: Use the same API and use-case wiring as production; choose the Redis or in-memory infrastructure appropriate to the test
4. **Fast Tests**: Keep integration tests focused and fast (< 1 second per test)
5. **Error Messages**: Provide clear error messages that help diagnose failures

## Test Configuration

Integration tests use a test-specific configuration with:
- Tracing disabled
- Metrics disabled
- Log level set to INFO
- CORS set to allow all origins
- Random ports assigned by httptest.Server

See `getTestConfig()` in `server.go` for details.
