package state

import (
	"sync"
	"time"

	"agq-daemon/internal/domain"
)

// State is a thread-safe container for daemon status read by API handlers and
// updated by the poller.
type State struct {
	mu          sync.RWMutex
	daemonState domain.DaemonState
	emails      []string
	lastPollAt  *time.Time
	nextPollAt  *time.Time
	startedAt   time.Time
}

// New creates an idle daemon state.
func New() *State {
	return &State{
		daemonState: domain.StateIdle,
		startedAt:   time.Now(),
	}
}

// SetActive marks the daemon active and records the active account emails.
func (s *State) SetActive(emails []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.daemonState = domain.StateActive
	s.emails = append([]string(nil), emails...)
}

// SetIdle marks the daemon idle and clears active account and next-poll data.
func (s *State) SetIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.daemonState = domain.StateIdle
	s.emails = nil
	s.nextPollAt = nil
}

// SetPollTimes records the last completed poll and the expected next poll.
func (s *State) SetPollTimes(last, next time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	last = last.UTC()
	next = next.UTC()
	s.lastPollAt = &last
	s.nextPollAt = &next
}

// Snapshot returns a consistent copy of the daemon status.
func (s *State) Snapshot() domain.DaemonStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return domain.DaemonStatus{
		State:      s.daemonState,
		Emails:     append([]string(nil), s.emails...),
		Uptime:     time.Since(s.startedAt).Round(time.Second).String(),
		StartedAt:  s.startedAt,
		LastPollAt: cloneTime(s.lastPollAt),
		NextPollAt: cloneTime(s.nextPollAt),
	}
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}
