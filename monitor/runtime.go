// Package monitor wires discovery, quota polling, persistence, and the local
// API into one lifecycle hosted by either the daemon or desktop application.
package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"sync"

	"agq-daemon/internal/api"
	"agq-daemon/internal/detector"
	"agq-daemon/internal/domain"
	"agq-daemon/internal/languageserver"
	"agq-daemon/internal/poller"
	"agq-daemon/internal/state"
	"agq-daemon/internal/store"
)

// Config describes one embedded monitoring runtime.
type Config struct {
	DataDir string
	Addr    string
	Logger  *slog.Logger
}

// Runtime owns all long-lived monitor resources.
type Runtime struct {
	cfg Config

	mu       sync.Mutex
	db       *store.DB
	listener net.Listener
	cancel   context.CancelFunc
	done     chan error
}

// Addr returns the bound API address after Start, or an empty string before
// startup and after shutdown.
func (r *Runtime) Addr() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listener == nil {
		return ""
	}
	return r.listener.Addr().String()
}

// New validates the runtime configuration. Resources are opened by Start.
func New(cfg Config) (*Runtime, error) {
	if cfg.DataDir == "" {
		return nil, errors.New("monitor data directory is required")
	}
	if cfg.Addr == "" {
		return nil, errors.New("monitor listen address is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Runtime{cfg: cfg}, nil
}

// Start opens storage and begins discovery, polling, and API serving.
func (r *Runtime) Start(parent context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return errors.New("monitor runtime already started")
	}

	db, err := store.Open(filepath.Join(r.cfg.DataDir, "agq.db"))
	if err != nil {
		return fmt.Errorf("open monitor database: %w", err)
	}
	listener, err := net.Listen("tcp", r.cfg.Addr)
	if err != nil {
		db.Close()
		return fmt.Errorf("listen on %s: %w", r.cfg.Addr, err)
	}

	ctx, cancel := context.WithCancel(parent)
	appState := state.New()
	infoCh := make(chan []*domain.ProcessInfo, 1)
	scanner := detector.New(languageserver.NewClient(languageserver.ProbeTimeout), r.cfg.Logger)
	quotaPoller := poller.New(db, appState, languageserver.NewClient(languageserver.RequestTimeout), poller.Config{Logger: r.cfg.Logger})
	server := api.New(db, appState, api.WithLogger(r.cfg.Logger))
	done := make(chan error, 1)

	r.db = db
	r.listener = listener
	r.cancel = cancel
	r.done = done

	go scanner.Run(ctx, infoCh)
	go quotaPoller.Run(ctx, infoCh)
	go func() {
		done <- server.Serve(ctx, listener)
		close(done)
	}()

	r.cfg.Logger.Info("monitor runtime started", "addr", listener.Addr().String())
	return nil
}

// Stop gracefully shuts down owned resources. It is safe to call repeatedly.
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if r.cancel == nil {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	done := r.done
	db := r.db
	r.cancel = nil
	r.done = nil
	r.listener = nil
	r.db = nil
	r.mu.Unlock()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			r.cfg.Logger.Warn("monitor API stopped with an error", "err", err)
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close monitor database: %w", err)
	}
	r.cfg.Logger.Info("monitor runtime stopped")
	return nil
}
