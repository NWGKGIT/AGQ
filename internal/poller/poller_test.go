package poller

import (
	"context"
	"sync"
	"testing"
	"time"

	"agq-daemon/internal/domain"
)

func TestRunPollsImmediately(t *testing.T) {
	store := &memoryStore{}
	status := &memoryStatus{}
	fetcher := &fakeFetcher{snapshots: map[int]*domain.QuotaSnapshot{
		1: {Email: "user@example.com", CapturedAt: time.Now()},
	}}
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

	cancel()
	<-done
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

type memoryStatus struct{}

func (m *memoryStatus) SetActive([]string)                {}
func (m *memoryStatus) SetIdle()                          {}
func (m *memoryStatus) SetPollTimes(last, next time.Time) {}

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
