package poller

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"agq-daemon/internal/domain"
)

const (
	// DefaultNormalInterval is the normal polling cadence while active.
	DefaultNormalInterval = 60 * time.Second

	// DefaultBackoffInterval is used after consecutive all-failure cycles.
	DefaultBackoffInterval = 5 * time.Minute

	// DefaultMaxFailures is the number of all-failure cycles before backoff.
	DefaultMaxFailures = 5

	// AuthSettleDelay is how long the poller waits before its first poll of
	// a newly detected process set while already ACTIVE. This gives a freshly
	// spawned language server time to finish its OAuth handshake so the
	// poller doesn't capture stale session data.
	AuthSettleDelay = 25 * time.Second
)

// SnapshotStore persists quota snapshots.
type SnapshotStore interface {
	SaveSnapshot(domain.QuotaSnapshot) error
}

// StatusWriter updates daemon status.
type StatusWriter interface {
	SetActive(emails []string)
	SetIdle()
	SetPollTimes(last, next time.Time)
}

// SnapshotFetcher fetches one quota snapshot from a detected language server.
type SnapshotFetcher interface {
	FetchSnapshot(ctx context.Context, info *domain.ProcessInfo) (*domain.QuotaSnapshot, error)
}

// Config controls poller timing and logging.
type Config struct {
	NormalInterval  time.Duration
	BackoffInterval time.Duration
	MaxFailures     int
	AuthSettleDelay time.Duration
	Logger          *slog.Logger
}

// Poller consumes detector updates and stores quota snapshots.
type Poller struct {
	store   SnapshotStore
	state   StatusWriter
	fetcher SnapshotFetcher
	config  Config
}

// New creates a Poller.
func New(store SnapshotStore, state StatusWriter, fetcher SnapshotFetcher, config Config) *Poller {
	return &Poller{
		store:   store,
		state:   state,
		fetcher: fetcher,
		config:  config.withDefaults(),
	}
}

// Run watches detector updates until ctx is cancelled or the info channel is
// closed.
func (p *Poller) Run(ctx context.Context, info <-chan []*domain.ProcessInfo) {
	var (
		current         []*domain.ProcessInfo
		failures        int
		pollTicker      *time.Ticker
		pollCh          <-chan time.Time
		currentInterval = p.config.NormalInterval
		authSettleTimer *time.Timer
		authSettleCh    <-chan time.Time
	)

	stopTicker := func() {
		if pollTicker != nil {
			pollTicker.Stop()
			pollTicker = nil
			pollCh = nil
		}
	}
	startTicker := func(d time.Duration) {
		stopTicker()
		currentInterval = d
		pollTicker = time.NewTicker(d)
		pollCh = pollTicker.C
	}
	stopAuthSettle := func() {
		if authSettleTimer != nil {
			authSettleTimer.Stop()
			authSettleTimer = nil
			authSettleCh = nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			stopTicker()
			stopAuthSettle()
			return

		case update, ok := <-info:
			if !ok {
				stopTicker()
				stopAuthSettle()
				return
			}
			if len(update) == 0 {
				if len(current) > 0 {
					p.logger().Info("poller: all language servers gone; going idle")
					p.state.SetIdle()
					stopTicker()
					stopAuthSettle()
					failures = 0
				}
				current = nil
				continue
			}

			wasEmpty := len(current) == 0
			pidsChanged := !wasEmpty && pidSetChanged(current, update)
			current = update
			switch {
			case wasEmpty:
				p.logger().Info("poller: language server(s) detected; going active", "count", len(update))
				failures = 0
				startTicker(p.config.NormalInterval)
				p.pollAllProcesses(ctx, current, &failures, startTicker, currentInterval)
			case pidsChanged:
				// A new set of PIDs while already active means Antigravity
				// spawned a fresh language server (e.g. an account switch).
				// Wait for it to finish its OAuth handshake before polling,
				// so we don't capture stale session data from a server that
				// hasn't authenticated yet.
				p.logger().Info("poller: process set changed", "count", len(update))
				p.logger().Info("poller: waiting for language server auth", "delay", p.config.AuthSettleDelay)
				failures = 0
				stopTicker()
				stopAuthSettle()
				authSettleTimer = time.NewTimer(p.config.AuthSettleDelay)
				authSettleCh = authSettleTimer.C
			default:
				p.logger().Debug("poller: process list refreshed", "count", len(update))
			}

		case <-authSettleCh:
			authSettleTimer = nil
			authSettleCh = nil
			startTicker(p.config.NormalInterval)
			p.pollAllProcesses(ctx, current, &failures, startTicker, currentInterval)

		case <-pollCh:
			if len(current) == 0 {
				continue
			}
			if failures >= p.config.MaxFailures {
				currentInterval = p.config.BackoffInterval
			} else {
				currentInterval = p.config.NormalInterval
			}
			p.pollAllProcesses(ctx, current, &failures, startTicker, currentInterval)
		}
	}
}

func (p *Poller) pollAllProcesses(
	ctx context.Context,
	infos []*domain.ProcessInfo,
	failures *int,
	startTicker func(time.Duration),
	currentInterval time.Duration,
) {
	results := make(map[string]*domain.QuotaSnapshot, len(infos))

	for _, info := range infos {
		snap, err := p.fetcher.FetchSnapshot(ctx, info)
		if err != nil {
			p.logger().Warn("poller: poll failed",
				"pid", info.Pid,
				"port", info.ActivePort,
				"err", err)
			continue
		}
		results[snap.Email] = snap
	}

	if len(results) == 0 {
		*failures++
		p.logger().Warn("poller: all polls failed", "consecutive_failures", *failures)
		if *failures >= p.config.MaxFailures {
			p.logger().Warn("poller: backing off", "interval", p.config.BackoffInterval)
			startTicker(p.config.BackoffInterval)
		}
		return
	}

	nextInterval := currentInterval
	if *failures >= p.config.MaxFailures {
		p.logger().Info("poller: polls recovered; resuming normal interval")
		startTicker(p.config.NormalInterval)
		nextInterval = p.config.NormalInterval
	}
	*failures = 0

	emails := make([]string, 0, len(results))
	for email, snap := range results {
		if err := p.store.SaveSnapshot(*snap); err != nil {
			p.logger().Warn("poller: save snapshot failed", "email", email, "err", err)
			continue
		}
		emails = append(emails, email)
		p.logger().Info("poller: snapshot saved",
			"email", email,
			"prompt_avail", snap.PromptCreditsAvailable,
			"flow_avail", snap.FlowCreditsAvailable,
			"models", len(snap.Models))
	}

	sort.Strings(emails)
	now := time.Now()
	p.state.SetActive(emails)
	p.state.SetPollTimes(now, now.Add(nextInterval))
}

// pidSetChanged reports whether the set of PIDs in next differs from prev,
// ignoring ordering. A differing length or any PID not present in both is
// treated as a change.
func pidSetChanged(prev, next []*domain.ProcessInfo) bool {
	if len(prev) != len(next) {
		return true
	}
	prevPids := make([]int, len(prev))
	nextPids := make([]int, len(next))
	for i := range prev {
		prevPids[i] = prev[i].Pid
	}
	for i := range next {
		nextPids[i] = next[i].Pid
	}
	sort.Ints(prevPids)
	sort.Ints(nextPids)
	for i := range prevPids {
		if prevPids[i] != nextPids[i] {
			return true
		}
	}
	return false
}

func (c Config) withDefaults() Config {
	if c.NormalInterval <= 0 {
		c.NormalInterval = DefaultNormalInterval
	}
	if c.BackoffInterval <= 0 {
		c.BackoffInterval = DefaultBackoffInterval
	}
	if c.MaxFailures <= 0 {
		c.MaxFailures = DefaultMaxFailures
	}
	if c.AuthSettleDelay <= 0 {
		c.AuthSettleDelay = AuthSettleDelay
	}
	return c
}

func (p *Poller) logger() *slog.Logger {
	if p == nil || p.config.Logger == nil {
		return slog.Default()
	}
	return p.config.Logger
}
