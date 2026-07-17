package main

import (
	"context"
	"sync"

	"agq-desktop/internal/apiclient"
	"agq-desktop/internal/config"
)

// App is the Wails application context. Bound methods exposed to the
// frontend live on this struct.
type App struct {
	ctx context.Context

	mu     sync.RWMutex
	cfg    config.Config
	client *apiclient.Client
}

// NewApp creates a new App application struct.
func NewApp() *App {
	cfg := config.Load()
	return &App{
		cfg:    cfg,
		client: apiclient.New(cfg.Port),
	}
}

// startup saves the runtime context so bound methods can call Wails runtime
// functions.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

// SetConfig persists new desktop settings and reconnects the API client.
func (a *App) SetConfig(cfg config.Config) (config.Config, error) {
	if err := config.Save(cfg); err != nil {
		return a.GetConfig(), err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.client = apiclient.New(cfg.Port)
	a.mu.Unlock()
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
