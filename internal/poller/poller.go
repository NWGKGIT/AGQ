package poller

import (
	"context"
	"log/slog"
	"time"

	"agq-daemon/internal/domain"
)

const (
	DefaultNormalInterval  = 60 * time.Second
	DefaultBackoffInterval = 5 * time.Minute
	DefaultMaxFailures     = 5
)

type SnapshotStore interface {
	SaveSnapshot(domain.QuotaSnapshot) error
}

type StatusWriter interface {
	SetActive(emails []string)
	SetIdle()
	SetPollTimes(last, next time.Time)
}

type SnapshotFetcher interface {
	FetchSnapshot(ctx context.Context, info *domain.ProcessInfo) (*domain.QuotaSnapshot, error)
}

type Config struct {
	NormalInterval  time.Duration
	BackoffInterval time.Duration
	MaxFailures     int
	Logger          *slog.Logger
}

type Poller struct {
	store   SnapshotStore
	state   StatusWriter
	fetcher SnapshotFetcher
	config  Config
}

func New(store SnapshotStore, state StatusWriter, fetcher SnapshotFetcher, config Config) *Poller {
	return &Poller{store: store, state: state, fetcher: fetcher, config: config.withDefaults()}
}

func (p *Poller) Run(ctx context.Context, info <-chan []*domain.ProcessInfo) {
	var current *domain.ProcessInfo
	ticker := time.NewTicker(p.config.NormalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-info:
			if !ok {
				return
			}
			if len(update) == 0 {
				current = nil
				p.state.SetIdle()
				continue
			}
			current = update[0]
			p.poll(ctx, current)
		case <-ticker.C:
			if current != nil {
				p.poll(ctx, current)
			}
		}
	}
}

func (p *Poller) poll(ctx context.Context, info *domain.ProcessInfo) {
	snap, err := p.fetcher.FetchSnapshot(ctx, info)
	if err != nil {
		p.logger().Warn("poller: poll failed", "pid", info.Pid, "err", err)
		return
	}
	if err := p.store.SaveSnapshot(*snap); err != nil {
		p.logger().Warn("poller: save snapshot failed", "email", snap.Email, "err", err)
		return
	}
	now := time.Now()
	p.state.SetActive([]string{snap.Email})
	p.state.SetPollTimes(now, now.Add(p.config.NormalInterval))
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
	return c
}

func (p *Poller) logger() *slog.Logger {
	if p == nil || p.config.Logger == nil {
		return slog.Default()
	}
	return p.config.Logger
}
