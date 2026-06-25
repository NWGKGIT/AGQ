package poller

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"agq-daemon/internal/domain"
)

func TestRunPollsImmediatelyAndDeduplicatesByEmail(t *testing.T) {
	store := &memoryStore{}
	status := &memoryStatus{}
	fetcher := &fakeFetcher{
		snapshots: map[int]*domain.QuotaSnapshot{
			1: {Email: "b@example.com", CapturedAt: time.Now()},
			2: {Email: "a@example.com", CapturedAt: time.Now()},
			3: {Email: "a@example.com", CapturedAt: time.Now().Add(time.Second)},
		},
	}
	p := New(store, status, fetcher, Config{
		NormalInterval:  time.Hour,
		BackoffInterval: 2 * time.Hour,
		Logger:          slog.Default(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	infoCh := make(chan []*domain.ProcessInfo)
	done := make(chan struct{})
	go func() {
		p.Run(ctx, infoCh)
		close(done)
	}()

	infoCh <- []*domain.ProcessInfo{{Pid: 1}, {Pid: 2}, {Pid: 3}}
	waitFor(t, func() bool { return store.savedCount() == 2 })

	if got := store.savedCount(); got != 2 {
		t.Fatalf("len(saved) = %d, want 2", got)
	}
	if got := status.emails(); !slices.Equal(got, []string{"a@example.com", "b@example.com"}) {
		t.Fatalf("activeEmails = %#v, want sorted unique emails", got)
	}
	last, next := status.pollTimes()
	if last.IsZero() || next.IsZero() {
		t.Fatalf("poll times not recorded: %s %s", last, next)
	}

	cancel()
	<-done
}

func TestRunSetsIdleWhenProcessesDisappear(t *testing.T) {
	store := &memoryStore{}
	status := &memoryStatus{}
	fetcher := &fakeFetcher{
		snapshots: map[int]*domain.QuotaSnapshot{
			1: {Email: "user@example.com", CapturedAt: time.Now()},
		},
	}
	p := New(store, status, fetcher, Config{NormalInterval: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	infoCh := make(chan []*domain.ProcessInfo)
	done := make(chan struct{})
	go func() {
		p.Run(ctx, infoCh)
		close(done)
	}()

	infoCh <- []*domain.ProcessInfo{{Pid: 1}}
	waitFor(t, func() bool { return store.savedCount() == 1 })
	infoCh <- []*domain.ProcessInfo{}
	waitFor(t, status.isIdle)

	cancel()
	<-done
}

func TestRunRepollsWhenPidSetChanges(t *testing.T) {
	store := &memoryStore{}
	status := &memoryStatus{}
	fetcher := &fakeFetcher{
		snapshots: map[int]*domain.QuotaSnapshot{
			1: {Email: "old@example.com", CapturedAt: time.Now()},
			2: {Email: "new@example.com", CapturedAt: time.Now()},
		},
	}
	// Long interval so any re-poll must come from the PID change, not a tick.
	p := New(store, status, fetcher, Config{NormalInterval: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	infoCh := make(chan []*domain.ProcessInfo)
	done := make(chan struct{})
	go func() {
		p.Run(ctx, infoCh)
		close(done)
	}()

	infoCh <- []*domain.ProcessInfo{{Pid: 1}}
	waitFor(t, func() bool { return store.savedCount() == 1 })

	// Account switch: Antigravity spawns a new language server with a new PID.
	infoCh <- []*domain.ProcessInfo{{Pid: 2}}
	waitFor(t, func() bool { return store.savedCount() == 2 })
	waitFor(t, func() bool { return slices.Equal(status.emails(), []string{"new@example.com"}) })

	cancel()
	<-done
}

func TestRunDoesNotRepollWhenPidSetUnchanged(t *testing.T) {
	store := &memoryStore{}
	status := &memoryStatus{}
	fetcher := &fakeFetcher{
		snapshots: map[int]*domain.QuotaSnapshot{
			1: {Email: "user@example.com", CapturedAt: time.Now()},
		},
	}
	p := New(store, status, fetcher, Config{NormalInterval: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	infoCh := make(chan []*domain.ProcessInfo)
	done := make(chan struct{})
	go func() {
		p.Run(ctx, infoCh)
		close(done)
	}()

	infoCh <- []*domain.ProcessInfo{{Pid: 1}}
	waitFor(t, func() bool { return store.savedCount() == 1 })

	// Same PID arriving again is a plain refresh and must not trigger a poll.
	infoCh <- []*domain.ProcessInfo{{Pid: 1}}
	// Give the poller a chance to (incorrectly) re-poll before asserting.
	time.Sleep(20 * time.Millisecond)
	if got := store.savedCount(); got != 1 {
		t.Fatalf("savedCount = %d, want 1 (no re-poll on unchanged PID set)", got)
	}

	cancel()
	<-done
}

func TestPidSetChanged(t *testing.T) {
	info := func(pids ...int) []*domain.ProcessInfo {
		out := make([]*domain.ProcessInfo, len(pids))
		for i, pid := range pids {
			out[i] = &domain.ProcessInfo{Pid: pid}
		}
		return out
	}

	cases := []struct {
		name       string
		prev, next []*domain.ProcessInfo
		want       bool
	}{
		{"same single", info(1), info(1), false},
		{"different single", info(1), info(2), true},
		{"same set reordered", info(1, 2), info(2, 1), false},
		{"one pid differs", info(1, 2), info(1, 3), true},
		{"length grows", info(1), info(1, 2), true},
		{"length shrinks", info(1, 2), info(1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pidSetChanged(tc.prev, tc.next); got != tc.want {
				t.Fatalf("pidSetChanged = %v, want %v", got, tc.want)
			}
		})
	}
}

type memoryStore struct {
	mu    sync.Mutex
	saved []domain.QuotaSnapshot
}

func (m *memoryStore) SaveSnapshot(snapshot domain.QuotaSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saved = append(m.saved, snapshot)
	return nil
}

func (m *memoryStore) savedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.saved)
}

type memoryStatus struct {
	mu           sync.Mutex
	activeEmails []string
	idle         bool
	lastPollAt   time.Time
	nextPollAt   time.Time
}

func (m *memoryStatus) SetActive(emails []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeEmails = append([]string(nil), emails...)
	m.idle = false
}

func (m *memoryStatus) SetIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idle = true
	m.activeEmails = nil
}

func (m *memoryStatus) SetPollTimes(last, next time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastPollAt = last
	m.nextPollAt = next
}

func (m *memoryStatus) emails() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]string(nil), m.activeEmails...)
}

func (m *memoryStatus) isIdle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.idle
}

func (m *memoryStatus) pollTimes() (time.Time, time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.lastPollAt, m.nextPollAt
}

type fakeFetcher struct {
	snapshots map[int]*domain.QuotaSnapshot
}

func (f *fakeFetcher) FetchSnapshot(ctx context.Context, info *domain.ProcessInfo) (*domain.QuotaSnapshot, error) {
	return f.snapshots[info.Pid], nil
}

func waitFor(t *testing.T, predicate func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}
