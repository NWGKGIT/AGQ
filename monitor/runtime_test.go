package monitor

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRuntimeStartHealthAndStop(t *testing.T) {
	runtime, err := New(Config{
		DataDir: t.TempDir(),
		Addr:    "127.0.0.1:0",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox does not permit loopback listeners: %v", err)
		}
		t.Fatalf("Start() error = %v", err)
	}
	if runtime.Addr() == "" {
		t.Fatal("Addr() is empty after Start")
	}

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + runtime.Addr() + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health error = %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", response.StatusCode)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runtime.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if runtime.Addr() != "" {
		t.Fatalf("Addr() = %q after Stop, want empty", runtime.Addr())
	}
}

func TestNewRequiresDataDir(t *testing.T) {
	if _, err := New(Config{Addr: "127.0.0.1:0"}); err == nil {
		t.Fatal("New() without data directory succeeded")
	}
}

func TestRuntimeWithoutAddrServesInProcess(t *testing.T) {
	runtime, err := New(Config{
		DataDir: t.TempDir(),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if runtime.Addr() != "" {
		t.Fatalf("Addr() = %q without a configured address, want empty", runtime.Addr())
	}

	handler := runtime.Handler()
	if handler == nil {
		t.Fatal("Handler() is nil after Start")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", rec.Code)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runtime.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if runtime.Handler() != nil {
		t.Fatal("Handler() is non-nil after Stop, want nil")
	}
}
