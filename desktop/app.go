package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"agq-daemon/monitor"
	"agq-desktop/internal/apiclient"
	"agq-desktop/internal/config"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails application context. Bound methods exposed to the
// frontend live on this struct.
type App struct {
	ctx context.Context

	mu      sync.RWMutex
	cfg     config.Config
	client  *apiclient.Client
	runtime *monitor.Runtime
}

// NewApp creates a new App application struct.
func NewApp() *App {
	app := &App{cfg: config.Load()}
	// The client resolves the handler per request, so it stays valid across
	// monitor restarts and reports "unreachable" while the monitor is down.
	app.client = apiclient.New(app.monitorHandler)
	return app
}

// monitorHandler returns the embedded monitor's API handler, or nil while the
// monitor is not running.
func (a *App) monitorHandler() http.Handler {
	a.mu.RLock()
	runtime := a.runtime
	a.mu.RUnlock()
	if runtime == nil {
		return nil
	}
	return runtime.Handler()
}

// startup saves the runtime context so bound methods can call Wails runtime
// functions.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.startMonitor(ctx); err != nil {
		slog.Error("embedded monitor failed to start", "err", err)
	}
}

// shutdown stops the embedded monitor before Wails tears down the process.
func (a *App) shutdown(_ context.Context) {
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.stopMonitor(stopCtx); err != nil {
		slog.Warn("embedded monitor shutdown failed", "err", err)
	}
}

func (a *App) startMonitor(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	configPath, err := config.Path()
	if err != nil {
		return err
	}
	// The app reaches the monitor in-process; a TCP listener is only bound
	// when the user opts into exposing the API to external tools.
	addr := ""
	if a.cfg.ExposeAPI {
		addr = fmt.Sprintf("127.0.0.1:%d", a.cfg.Port)
	}
	runtime, err := monitor.New(monitor.Config{
		DataDir: filepath.Dir(configPath),
		Addr:    addr,
		Logger:  slog.Default(),
		OnUpdate: func() {
			if a.ctx != nil {
				wailsruntime.EventsEmit(a.ctx, "agq:data-updated")
			}
		},
	})
	if err != nil {
		return err
	}
	if err := runtime.Start(ctx); err != nil {
		return err
	}
	a.runtime = runtime
	return nil
}

func (a *App) stopMonitor(ctx context.Context) error {
	a.mu.Lock()
	runtime := a.runtime
	a.runtime = nil
	a.mu.Unlock()
	if runtime == nil {
		return nil
	}
	return runtime.Stop(ctx)
}

func (a *App) api() *apiclient.Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// GetConfig returns the current desktop settings.
func (a *App) GetConfig() config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// SetConfig persists new desktop settings and restarts the embedded monitor
// when the API exposure settings changed.
func (a *App) SetConfig(cfg config.Config) (config.Config, error) {
	previous := a.GetConfig()
	if err := config.Save(cfg); err != nil {
		return a.GetConfig(), err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()

	listenerChanged := previous.ExposeAPI != cfg.ExposeAPI ||
		(cfg.ExposeAPI && previous.Port != cfg.Port)
	if listenerChanged && a.ctx != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := a.stopMonitor(stopCtx)
		cancel()
		if err != nil {
			return cfg, err
		}
		if err := a.startMonitor(a.ctx); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// GetHealth calls the daemon's health endpoint.
func (a *App) GetHealth() (apiclient.Health, error) {
	return a.api().Health()
}

// GetStatus returns the daemon's live status.
func (a *App) GetStatus() (apiclient.DaemonStatus, error) {
	return a.api().Status()
}

// GetAccounts returns all known accounts with their latest snapshots.
func (a *App) GetAccounts() (apiclient.AccountsResponse, error) {
	return a.api().Accounts()
}

// GetCurrentAccount returns the account(s) currently logged in.
func (a *App) GetCurrentAccount() (apiclient.CurrentAccount, error) {
	return a.api().CurrentAccount()
}

// GetLatestSnapshot returns the newest snapshot for an account.
func (a *App) GetLatestSnapshot(email string) (apiclient.Snapshot, error) {
	return a.api().LatestSnapshot(email)
}

// GetSnapshots returns snapshot history, newest first.
func (a *App) GetSnapshots(email string, limit int, before string) (apiclient.SnapshotsResponse, error) {
	return a.api().Snapshots(email, limit, before)
}

// GetSparklines returns 7-day per-model series for an account.
func (a *App) GetSparklines(email string) (apiclient.SparklinesResponse, error) {
	return a.api().Sparklines(email)
}

// GetTimeline returns inferred login/logout events for an account.
func (a *App) GetTimeline(email string) (apiclient.TimelineResponse, error) {
	return a.api().Timeline(email)
}

// GetModelsLatest returns model quotas from each account's newest snapshot.
func (a *App) GetModelsLatest() (apiclient.ModelsLatestResponse, error) {
	return a.api().ModelsLatest()
}

// GetAccountModels returns each model's newest known quota for one account.
func (a *App) GetAccountModels(email string) (apiclient.AccountModelsResponse, error) {
	return a.api().AccountModels(email)
}

// GetTimeseries returns per-day provider aggregates.
func (a *App) GetTimeseries(rangeKey, agg string) (apiclient.TimeseriesResponse, error) {
	return a.api().Timeseries(rangeKey, agg)
}

// GetBreakdown returns the account x model consumption table.
func (a *App) GetBreakdown() (apiclient.BreakdownResponse, error) {
	return a.api().Breakdown()
}

// GetStats returns the analytics headline figures.
func (a *App) GetStats() (apiclient.Stats, error) {
	return a.api().Stats()
}
