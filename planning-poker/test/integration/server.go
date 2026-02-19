package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"planning-poker/internal/config"
	"planning-poker/internal/infra/boundaries/bus/redis"
	"planning-poker/internal/setup"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type TestServer struct {
	Server    *httptest.Server
	Router    *mux.Router
	Container *setup.Container
	config    *config.Config
}

func NewTestServer(t *testing.T) *TestServer {
	t.Helper()
	cfg := getTestConfig()
	setup.ConfigureLogging(cfg)

	container := setup.NewContainer(cfg)

	r := mux.NewRouter()
	setup.ConfigureAPIs(r, container)
	ts := httptest.NewServer(r)

	return &TestServer{
		Server:    ts,
		Router:    r,
		Container: container,
		config:    cfg,
	}
}

func (ts *TestServer) Close() {
	ts.Server.Close()
	if hub, ok := ts.Container.Hub.(*redis.RedisHub); ok {
		_ = hub.Close()
	}

	ts.cleanRedis()
	time.Sleep(100 * time.Millisecond)
}

func (ts *TestServer) cleanRedis() {
	if err := ts.Container.Infra.RedisClient.FlushDB(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("failed to flush redis: %v", err))
	}
}

func (ts *TestServer) GetJSON(t *testing.T, path string, target any) (*http.Response, error) {
	t.Helper()

	resp, err := http.Get(ts.Server.URL + path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return resp, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return resp, nil
}

func getTestConfig() *config.Config {
	cfg, err := config.LoadTestConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load test config: %v", err))
	}
	return cfg
}
